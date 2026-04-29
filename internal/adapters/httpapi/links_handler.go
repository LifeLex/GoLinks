package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golinks/internal/core/links"
	"golinks/internal/platform/config"
	"golinks/internal/platform/logger"

	"github.com/gorilla/mux"
)

// LinkService is the inbound port consumed by this handler. The
// internal/core/links.Service is the production implementation; tests pass a
// mock with the same surface.
type LinkService interface {
	GetLink(ctx context.Context, word, searchTerm string) (string, error)
	UpdateLink(ctx context.Context, req links.LinkRequest, userID string) error
	GetRecentQueries(ctx context.Context) ([]links.PopularQuery, error)
	GetAllKeywords(ctx context.Context) ([]links.KeywordInfo, error)
}

// LinksHandler exposes the golink redirect plus the JSON CRUD API.
type LinksHandler struct {
	service LinkService
	config  *config.Config
	logger  *logger.Logger
}

// NewLinksHandler wires the handler against the service port.
func NewLinksHandler(service LinkService, cfg *config.Config, log *logger.Logger) *LinksHandler {
	return &LinksHandler{service: service, config: cfg, logger: log}
}

// Register registers all link-related routes on the given router.
func (h *LinksHandler) Register(router *mux.Router) {
	// The golink contract: server-side 302 so browser search-engine integrations work.
	router.HandleFunc("/query/{path:.*}", h.Redirect).Methods(http.MethodGet)

	// JSON API.
	router.HandleFunc("/api/links", h.List).Methods(http.MethodGet)
	router.HandleFunc("/api/links", h.Create).Methods(http.MethodPost)

	// Back-compat alias from the template era.
	router.HandleFunc("/update/", h.UpdateLegacy).Methods(http.MethodPost)
}

// Redirect resolves /query/<word> to a 302.
func (h *LinksHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	queryPath := strings.TrimSuffix(mux.Vars(r)["path"], "/")
	userID := userIDFromRequest(r)

	h.logger.Info("Processing golink redirect: %s (user: %s)", queryPath, userID)

	targetURL, err := h.service.GetLink(r.Context(), queryPath, "")
	if err != nil {
		var invalid links.InvalidQueryError
		if errors.As(err, &invalid) {
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

// List returns everything the homepage needs in one call.
func (h *LinksHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	recent, err := h.service.GetRecentQueries(ctx)
	if err != nil {
		h.logger.Error("Failed to get recent queries: %v", err)
		recent = []links.PopularQuery{}
	}

	keywords, err := h.service.GetAllKeywords(ctx)
	if err != nil {
		h.logger.Error("Failed to get all keywords: %v", err)
		keywords = []links.KeywordInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"keywords":       keywords,
		"recent_queries": recent,
		"base_url":       h.config.BaseURL,
	})
}

// Create accepts a JSON {"word":"…","link":"…"} body.
func (h *LinksHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req links.LinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	req.Word = strings.TrimSpace(req.Word)
	req.Link = strings.TrimSpace(req.Link)

	if err := h.service.UpdateLink(r.Context(), req, userIDFromRequest(r)); err != nil {
		var invalid links.InvalidQueryError
		if errors.As(err, &invalid) {
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

// UpdateLegacy preserves the old /update/ form-encoded endpoint so existing
// browser search-engine configs keep working.
func (h *LinksHandler) UpdateLegacy(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	req := links.LinkRequest{
		Word: strings.TrimSpace(r.FormValue("word")),
		Link: strings.TrimSpace(r.FormValue("link")),
	}
	if err := h.service.UpdateLink(r.Context(), req, userIDFromRequest(r)); err != nil {
		var invalid links.InvalidQueryError
		if errors.As(err, &invalid) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("Link added successfully!"))
}
