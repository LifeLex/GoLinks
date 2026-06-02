package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golinks/internal/logger"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

// DocumentHandler exposes the docs CRUD as JSON.
// All server-side HTML rendering has been removed — the React client does
// MDX compilation in the browser via @mdx-js/mdx.
type DocumentHandler struct {
	docService *service.DocumentService
	logger     *logger.Logger
}

// NewDocumentHandler creates a new document handler.
func NewDocumentHandler(docService *service.DocumentService, log *logger.Logger) *DocumentHandler {
	log.Info("Document handler initialized")
	return &DocumentHandler{
		docService: docService,
		logger:     log,
	}
}

// RegisterRoutes wires the /api/docs endpoints. Reads are public; uploads and
// deletes are admin-only — runtime MDX evaluates JSX in the viewer's browser,
// so write access must be restricted.
func (h *DocumentHandler) RegisterRoutes(public, admin *mux.Router) {
	public.HandleFunc("/api/docs", h.ListDocuments).Methods("GET")
	public.HandleFunc("/api/docs/{filename}", h.GetDocument).Methods("GET")
	admin.HandleFunc("/api/docs", h.UploadDocument).Methods("POST")
	admin.HandleFunc("/api/docs/{filename}", h.DeleteDocument).Methods("DELETE")
}

// GetDocument returns the raw source and metadata of a single document.
func (h *DocumentHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Accept both "sample" and "sample.md"; try .md then .mdx when no extension.
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".mdx") {
		if doc, err := h.docService.GetDocument(r.Context(), filename+".md"); err == nil {
			writeJSON(w, http.StatusOK, doc)
			return
		}
		filename = filename + ".mdx"
	}

	doc, err := h.docService.GetDocument(r.Context(), filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Document not found: %v", err), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// UploadDocument persists an uploaded .md/.mdx file. Admin-gated at the route
// level (see RegisterRoutes) because runtime MDX evaluates JSX in the browser.
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
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

	if err := h.docService.SaveDocument(r.Context(), filename, file); err != nil {
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

// ListDocuments returns metadata for every document on disk.
func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.docService.ListDocuments(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"documents": docs})
}

// DeleteDocument removes a document.
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}
	if err := h.docService.DeleteDocument(r.Context(), filename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete document: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Document deleted successfully",
	})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
