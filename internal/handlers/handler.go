package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"golinks/internal/config"
	"golinks/internal/domain"
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
}

// NewHandler creates a new handler
func NewHandler(linkService LinkService, cfg *config.Config) *Handler {
	// Load templates
	templates := template.Must(template.New("").Funcs(template.FuncMap{
		"urlify": func(url string) template.HTML {
			if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
				return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, url, url))
			}
			return template.HTML(url)
		},
	}).ParseGlob("web/templates/*.html"))

	return &Handler{
		linkService: linkService,
		config:      cfg,
		templates:   templates,
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
}

// RedirectHandler handles golink redirects
func (h *Handler) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	queryPath := vars["path"]
	queryPath = strings.TrimSuffix(queryPath, "/")

	userID := h.getUserID(r)

	targetURL, err := h.linkService.GetLink(ctx, queryPath, "")
	if err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			// Redirect to homepage with missing query parameter
			redirectURL := fmt.Sprintf("%s/homepage/?missing=%s", h.config.BaseURL, queryPath)
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("query word=%s user=%s response=%s", queryPath, userID, targetURL)
	http.Redirect(w, r, targetURL, http.StatusFound)
}

// UpdateLinkHandler handles link creation/updates
func (h *Handler) UpdateLinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.LinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	userID := h.getUserID(r)

	if err := h.linkService.UpdateLink(ctx, req, userID); err != nil {
		if _, ok := err.(service.InvalidQueryError); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"detail": err.Error()})
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("update word=%s user=%s link=%s", req.Word, userID, req.Link)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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

	// Get recent queries and keywords
	recentQueries, err := h.linkService.GetRecentQueries(ctx)
	if err != nil {
		log.Printf("Failed to get recent queries: %v", err)
		recentQueries = []domain.PopularQuery{}
	}

	allKeywords, err := h.linkService.GetAllKeywords(ctx)
	if err != nil {
		log.Printf("Failed to get all keywords: %v", err)
		allKeywords = []domain.KeywordInfo{}
	}

	log.Printf("homepage user=%s", userID)

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
		log.Printf("Failed to execute template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SetupHandler handles the setup page
func (h *Handler) SetupHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	log.Printf("setup user=%s", userID)

	data := struct {
		BaseURL string
	}{
		BaseURL: h.config.BaseURL,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "setup.html", data); err != nil {
		log.Printf("Failed to execute template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getUserID extracts user ID from request (simplified - no OAuth2 for now)
func (h *Handler) getUserID(r *http.Request) string {
	// For now, return a default user. In production, this would extract from OAuth2 cookie
	return "DefaultUser"
}
