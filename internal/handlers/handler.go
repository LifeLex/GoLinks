package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/logger"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

// LinkService is the subset of the link service the HTTP layer depends on.
type LinkService interface {
	GetLink(ctx context.Context, word string, searchTerm string) (string, error)
	UpdateLink(ctx context.Context, req domain.LinkRequest, userID string) error
	GetRecentQueries(ctx context.Context) ([]domain.PopularQuery, error)
	GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error)
	Search(ctx context.Context, query string, limit int) ([]domain.KeywordInfo, error)
}

// Handler owns the redirect + JSON endpoints for links.
// The SPA fallback (serving index.html for unmatched routes) lives in the
// main package so it can reference the embedded frontend filesystem.
type Handler struct {
	linkService LinkService
	config      *config.Config
	logger      *logger.Logger
}

// NewHandler builds a new Handler. No templates are loaded — the UI is a
// React SPA served from the embedded frontend filesystem.
func NewHandler(linkService LinkService, cfg *config.Config, log *logger.Logger) *Handler {
	log.Info("Handler initialized (JSON + redirect only)")
	return &Handler{
		linkService: linkService,
		config:      cfg,
		logger:      log,
	}
}

// RegisterRoutes wires the redirect and JSON endpoints. Reads go on the public
// router; writes go on the auth-gated router. Static-asset and SPA fallback
// handling is registered separately in cmd/server/main.go.
func (h *Handler) RegisterRoutes(public, authed *mux.Router) {
	// Liveness/readiness probe target for orchestrators (Kubernetes, Docker).
	public.HandleFunc("/healthz", h.HealthCheck).Methods("GET")

	// The golink contract: server-side 302 so browser search-engine integrations work.
	public.HandleFunc("/query/{path:.*}", h.RedirectHandler).Methods("GET")

	// Public reads.
	public.HandleFunc("/api/links", h.ListLinks).Methods("GET")
	public.HandleFunc("/api/search", h.SearchLinks).Methods("GET")

	// Authenticated writes.
	authed.HandleFunc("/api/links", h.CreateLink).Methods("POST")

	// Legacy form-encoded create. Now requires auth — closing the old
	// unauthenticated write path. An authenticated admin's browser keyword POST
	// is same-origin and carries the session cookie, so it keeps working.
	authed.HandleFunc("/update/", h.UpdateLinkLegacy).Methods("POST")
}

// HealthCheck is a liveness/readiness probe target. It returns 200 as long as
// the HTTP server is serving requests; it intentionally does no database work so
// a transient query failure can't flap the pod.
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RedirectHandler resolves a golink and issues a 302.
func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryPath := strings.TrimSuffix(mux.Vars(r)["path"], "/")
	userID := h.getUserID(r)

	h.logger.Info("Processing golink redirect: %s (user: %s)", queryPath, userID)

	targetURL, err := h.linkService.GetLink(ctx, queryPath, "")
	if err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			h.logger.Warn("Invalid query '%s' - redirecting to home: %v", queryPath, err)
			http.Redirect(w, r, fmt.Sprintf("%s/?missing=%s", h.config.BaseURL, queryPath), http.StatusFound)
			return
		}
		h.logger.Error("Failed to get link for query '%s': %v", queryPath, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}

// ListLinks returns everything the homepage needs in one call.
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	recent, err := h.linkService.GetRecentQueries(ctx)
	if err != nil {
		h.logger.Error("Failed to get recent queries: %v", err)
		recent = []domain.PopularQuery{}
	}

	keywords, err := h.linkService.GetAllKeywords(ctx)
	if err != nil {
		h.logger.Error("Failed to get all keywords: %v", err)
		keywords = []domain.KeywordInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keywords":       keywords,
		"recent_queries": recent,
		"base_url":       h.config.BaseURL,
	})
}

// SearchLinks handles GET /api/search?q=&limit= and returns matching keywords.
func (h *Handler) SearchLinks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "limit must be an integer", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	results, err := h.linkService.Search(r.Context(), q, limit)
	if err != nil {
		h.logger.Error("Search failed for q='%s': %v", q, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"query":   q,
		"results": results,
	})
}

// CreateLink accepts a JSON {"word":"", "link":""} body.
func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req domain.LinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	req.Word = strings.TrimSpace(req.Word)
	req.Link = strings.TrimSpace(req.Link)

	if err := h.linkService.UpdateLink(r.Context(), req, h.getUserID(r)); err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			h.logger.Warn("Invalid link request word='%s': %v", req.Word, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.logger.Error("Failed to create link word='%s': %v", req.Word, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Link created: word='%s' link='%s'", req.Word, req.Link)
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// UpdateLinkLegacy preserves the old /update/ form endpoint so browser search
// engines that still POST form-encoded data keep working.
func (h *Handler) UpdateLinkLegacy(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	req := domain.LinkRequest{
		Word: strings.TrimSpace(r.FormValue("word")),
		Link: strings.TrimSpace(r.FormValue("link")),
	}
	if err := h.linkService.UpdateLink(r.Context(), req, h.getUserID(r)); err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("Link added successfully!"))
}

// getUserID returns the authenticated user's email, or "" when anonymous. Write
// handlers run behind RequireAuth, so a user is always present there; the
// public redirect handler may see "" (used only for logging).
func (h *Handler) getUserID(r *http.Request) string {
	if user := UserFromContext(r.Context()); user != nil {
		return user.Email
	}
	return ""
}
