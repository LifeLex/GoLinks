package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"golinks/internal/config"
	"golinks/internal/domain"
	"golinks/internal/logger"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

// LinkService interface for link operations
type LinkService interface {
	GetLink(ctx context.Context, word string, searchTerm string) (string, error)
	UpdateLink(ctx context.Context, req domain.LinkRequest, userID string) error
	GetRecentQueries(ctx context.Context) ([]domain.PopularQuery, error)
	GetAllKeywords(ctx context.Context) ([]domain.KeywordInfo, error)
}

// Handler holds the HTTP handlers
type Handler struct {
	linkService LinkService
	config      *config.Config
	templates   *template.Template
	logger      *logger.Logger
}

// NewHandler creates a new handler
func NewHandler(linkService LinkService, cfg *config.Config, log *logger.Logger) *Handler {
	log.Info("Loading HTML templates from web/templates/*.html")

	// Load templates
	templates := template.Must(template.New("").Funcs(template.FuncMap{
		"urlify": func(url string) template.HTML {
			if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
				return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, url, url))
			}
			return template.HTML(url)
		},
	}).ParseGlob("web/templates/*.html"))

	log.Info("Handler initialized successfully")

	return &Handler{
		linkService: linkService,
		config:      cfg,
		templates:   templates,
		logger:      log,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// API routes
	router.HandleFunc("/query/{path:.*}", h.RedirectHandler).Methods("GET")
	router.HandleFunc("/update/", h.UpdateLinkHandler).Methods("POST")
	router.HandleFunc("/homepage/", h.HomepageHandler).Methods("GET")
	router.HandleFunc("/setup/", h.SetupHandler).Methods("GET")

	// Root redirect to homepage
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/homepage/", http.StatusFound)
	}).Methods("GET")

	// 404 handler for all other routes
	router.NotFoundHandler = http.HandlerFunc(h.NotFoundHandler)
}

// RedirectHandler handles golink redirects
func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	queryPath := vars["path"]
	queryPath = strings.TrimSuffix(queryPath, "/")

	userID := h.getUserID(r)

	h.logger.Info("Processing golink redirect: %s (user: %s)", queryPath, userID)

	targetURL, err := h.linkService.GetLink(ctx, queryPath, "")
	if err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			h.logger.Warn("Invalid query '%s' - redirecting to homepage: %v", queryPath, err)
			// Redirect to homepage with missing query parameter
			redirectURL := fmt.Sprintf("%s/homepage/?missing=%s", h.config.BaseURL, queryPath)
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}

		h.logger.Error("Failed to get link for query '%s': %v", queryPath, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Redirecting '%s' to '%s' (user: %s)", queryPath, targetURL, userID)
	http.Redirect(w, r, targetURL, http.StatusFound)
}

// UpdateLinkHandler handles link creation/updates
func (h *Handler) UpdateLinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.LinkRequest

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Warn("Invalid form data in update request: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	req.Word = strings.TrimSpace(r.FormValue("word"))
	req.Link = strings.TrimSpace(r.FormValue("link"))

	h.logger.Info("Parsed form data: word='%s' link='%s'", req.Word, req.Link)

	userID := h.getUserID(r)

	h.logger.Info("Processing link update: word='%s' link='%s' user='%s'", req.Word, req.Link, userID)

	if err := h.linkService.UpdateLink(ctx, req, userID); err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			h.logger.Warn("Invalid link update request for word='%s': %v", req.Word, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		h.logger.Error("Failed to update link word='%s': %v", req.Word, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Link updated successfully: word='%s' link='%s' user='%s'", req.Word, req.Link, userID)

	// Return success message for HTMX
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte("Link added successfully!")); err != nil {
		h.logger.Error("Failed to write response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HomepageHandler handles the homepage
func (h *Handler) HomepageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := h.getUserID(r)

	// Get query parameters
	success := r.URL.Query().Get("success")
	failure := r.URL.Query().Get("failure")
	reason := r.URL.Query().Get("reason")
	missing := r.URL.Query().Get("missing")

	h.logger.Info("Rendering homepage for user '%s'", userID)
	if missing != "" {
		h.logger.Info("Homepage showing missing query: %s", missing)
	}

	// Get recent queries and keywords
	recentQueries, err := h.linkService.GetRecentQueries(ctx)
	if err != nil {
		h.logger.Error("Failed to get recent queries: %v", err)
		recentQueries = []domain.PopularQuery{}
	}

	allKeywords, err := h.linkService.GetAllKeywords(ctx)
	if err != nil {
		h.logger.Error("Failed to get all keywords: %v", err)
		allKeywords = []domain.KeywordInfo{}
	}

	h.logger.Debug("Homepage data loaded: %d recent queries, %d keywords", len(recentQueries), len(allKeywords))

	data := struct {
		Success       string
		Failure       string
		Reason        string
		Missing       string
		RecentQueries []domain.PopularQuery
		AllKeywords   []domain.KeywordInfo
		BaseURL       string
	}{
		Success:       success,
		Failure:       failure,
		Reason:        reason,
		Missing:       missing,
		RecentQueries: recentQueries,
		AllKeywords:   allKeywords,
		BaseURL:       h.config.BaseURL,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "homepage.html", data); err != nil {
		h.logger.Error("Failed to execute homepage template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Homepage rendered successfully")
}

// SetupHandler handles the setup page
func (h *Handler) SetupHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	h.logger.Info("Rendering setup page for user '%s'", userID)

	data := struct {
		BaseURL string
	}{
		BaseURL: h.config.BaseURL,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "setup.html", data); err != nil {
		h.logger.Error("Failed to execute setup template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Setup page rendered successfully")
}

// NotFoundHandler handles 404 errors
func (h *Handler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	h.logger.Info("404 page requested for path '%s' by user '%s'", r.URL.Path, userID)

	data := struct {
		BaseURL string
		Path    string
	}{
		BaseURL: h.config.BaseURL,
		Path:    r.URL.Path,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	if err := h.templates.ExecuteTemplate(w, "404.html", data); err != nil {
		h.logger.Error("Failed to execute 404 template: %v", err)
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	h.logger.Debug("404 page rendered successfully")
}

// getUserID extracts user ID from request (simplified - no OAuth2 for now)
func (h *Handler) getUserID(r *http.Request) string {
	// For now, return a default user. In production, this would extract from OAuth2 cookie
	return "DefaultUser"
}
