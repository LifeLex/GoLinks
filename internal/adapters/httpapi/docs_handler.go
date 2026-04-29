package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golinks/internal/core/docs"
	"golinks/internal/platform/logger"

	"github.com/gorilla/mux"
)

// DocsService is the inbound port consumed by this handler. The
// internal/core/docs.Service is the production implementation.
type DocsService interface {
	GetDocument(ctx context.Context, filename string) (*docs.DocumentSource, error)
	SaveDocument(ctx context.Context, filename string, content io.Reader) error
	ListDocuments(ctx context.Context) ([]docs.DocumentInfo, error)
	DeleteDocument(ctx context.Context, filename string) error
}

// DocsHandler exposes the /api/docs CRUD endpoints. All HTML rendering happens
// client-side; this handler returns raw markdown/MDX source.
type DocsHandler struct {
	service DocsService
	logger  *logger.Logger
}

// NewDocsHandler wires the handler.
func NewDocsHandler(service DocsService, log *logger.Logger) *DocsHandler {
	return &DocsHandler{service: service, logger: log}
}

// Register registers all /api/docs routes on the given router.
func (h *DocsHandler) Register(router *mux.Router) {
	router.HandleFunc("/api/docs", h.List).Methods(http.MethodGet)
	router.HandleFunc("/api/docs", h.Upload).Methods(http.MethodPost)
	router.HandleFunc("/api/docs/{filename}", h.Get).Methods(http.MethodGet)
	router.HandleFunc("/api/docs/{filename}", h.Delete).Methods(http.MethodDelete)
}

// Get returns the raw source and metadata of a single document.
func (h *DocsHandler) Get(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	doc, err := h.service.GetDocument(r.Context(), filename)
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			http.Error(w, fmt.Sprintf("Document not found: %s", filename), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Document not found: %v", err), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// Upload persists an uploaded .md/.mdx file. TODO: gate behind authentication
// before exposing publicly — runtime MDX evaluation makes this an XSS vector.
func (h *DocsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".mdx") {
		http.Error(w, "Only .md and .mdx files are allowed", http.StatusBadRequest)
		return
	}

	if err := h.service.SaveDocument(r.Context(), filename, file); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save document: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"filename": filename,
		"message":  "Document uploaded successfully",
		"url":      fmt.Sprintf("/docs/%s", filename),
	})
}

// List returns metadata for every document on disk.
func (h *DocsHandler) List(w http.ResponseWriter, r *http.Request) {
	docs, err := h.service.ListDocuments(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"documents": docs})
}

// Delete removes a document.
func (h *DocsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteDocument(r.Context(), filename); err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete document: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Document deleted successfully",
	})
}
