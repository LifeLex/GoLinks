package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"golinks/internal/logger"
	"golinks/internal/service"

	"github.com/gorilla/mux"
)

// DocumentHandler handles document-related HTTP requests
type DocumentHandler struct {
	docService *service.DocumentService
	logger     *logger.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(docService *service.DocumentService, log *logger.Logger) *DocumentHandler {
	log.Info("Document handler initialized")
	return &DocumentHandler{
		docService: docService,
		logger:     log,
	}
}

// RegisterRoutes registers document-related routes
func (h *DocumentHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/docs/{filename}", h.ServeDocument).Methods("GET")
	router.HandleFunc("/docs/", h.ListDocuments).Methods("GET")
	router.HandleFunc("/api/docs/upload", h.UploadDocument).Methods("POST")
	router.HandleFunc("/api/docs/{filename}", h.DeleteDocument).Methods("DELETE")
}

// ServeDocument renders and serves a document
func (h *DocumentHandler) ServeDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Add extension if not provided
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".mdx") {
		// Try .md first, then .mdx
		if result, err := h.docService.GetDocument(r.Context(), filename+".md"); err == nil {
			h.renderDocumentHTML(w, result)
			return
		}
		filename = filename + ".mdx"
	}

	result, err := h.docService.GetDocument(r.Context(), filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Document not found: %v", err), http.StatusNotFound)
		return
	}

	h.renderDocumentHTML(w, result)
}

// UploadDocument handles document upload
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	filename := header.Filename
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".mdx") {
		http.Error(w, "Only .md and .mdx files are allowed", http.StatusBadRequest)
		return
	}

	// Save document
	if err := h.docService.SaveDocument(r.Context(), filename, file); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save document: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":  true,
		"filename": filename,
		"message":  "Document uploaded successfully",
		"url":      fmt.Sprintf("/docs/%s", strings.TrimSuffix(filename, filepath.Ext(filename))),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// ListDocuments returns a list of available documents
func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.docService.ListDocuments(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": docs,
	})
}

// DeleteDocument removes a document
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	filename := vars["filename"]

	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	if err := h.docService.DeleteDocument(r.Context(), filename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete document: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// renderDocumentHTML renders the document using HTML template
func (h *DocumentHandler) renderDocumentHTML(w http.ResponseWriter, result *service.RenderResult) {
	// Create template data
	data := struct {
		Title       string
		Description string
		Type        string
		Content     template.HTML
		Metadata    map[string]interface{}
	}{
		Title:       result.Metadata.Title,
		Description: result.Metadata.Description,
		Type:        result.Metadata.Type,
		Content:     template.HTML(result.HTML),
		Metadata:    result.Metadata.Metadata,
	}

	// Simple HTML template (we'll create a proper template file later)
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - GoLinks Docs</title>
    <link rel="stylesheet" href="/static/styles.css">
    <!-- Prism.js for syntax highlighting -->
    <link href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism.min.css" rel="stylesheet" />
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js"></script>
    <style>
        .document-container {
            max-width: 800px;
            margin: 2rem auto;
            padding: 2rem;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .document-header {
            border-bottom: 1px solid #eee;
            padding-bottom: 1rem;
            margin-bottom: 2rem;
        }
        .document-type {
            display: inline-block;
            background: #f0f0f0;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-size: 0.8rem;
            text-transform: uppercase;
            margin-bottom: 0.5rem;
        }
        .document-content {
            line-height: 1.6;
        }
        .document-content h1, .document-content h2, .document-content h3 {
            margin-top: 2rem;
            margin-bottom: 1rem;
        }
        .document-content pre {
            background: #f6f8fa !important;
            padding: 1rem !important;
            border-radius: 6px !important;
            overflow-x: auto;
            border: 1px solid #d1d9e0 !important;
            margin: 1.5rem 0 !important;
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace !important;
            font-size: 0.875rem !important;
            line-height: 1.45 !important;
        }
        .document-content pre code {
            background: none !important;
            border: none !important;
            padding: 0 !important;
            margin: 0 !important;
            font-size: inherit !important;
            white-space: pre;
            word-wrap: normal;
            box-shadow: none !important;
        }
        /* Override Prism.js default styles */
        .document-content pre[class*="language-"] {
            background: #f6f8fa !important;
            border: 1px solid #d1d9e0 !important;
        }
        .document-content code[class*="language-"] {
            background: none !important;
        }
        .document-content code {
            background: #f0f0f0;
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
            font-size: 0.875rem;
            color: #d63384;
        }
        .document-content blockquote {
            border-left: 4px solid #ddd;
            margin: 1rem 0;
            padding-left: 1rem;
            color: #666;
        }
        .document-content table {
            width: 100%;
            border-collapse: collapse;
            margin: 1rem 0;
        }
        .document-content th, .document-content td {
            border: 1px solid #ddd;
            padding: 0.5rem;
            text-align: left;
        }
        .document-content th {
            background: #f8f8f8;
        }
        .alert {
            padding: 1rem;
            margin: 1rem 0;
            border-radius: 4px;
        }
        .alert-info {
            background: #e3f2fd;
            border: 1px solid #2196f3;
            color: #1976d2;
        }
    </style>
</head>
<body>
    <div class="document-container">
        <header class="document-header">
            <div class="document-type">{{.Type}}</div>
            <h1>{{.Title}}</h1>
            {{if .Description}}<p>{{.Description}}</p>{{end}}
        </header>
        <main class="document-content">
            {{.Content}}
        </main>
        <footer style="margin-top: 2rem; padding-top: 1rem; border-top: 1px solid #eee; color: #666; font-size: 0.9rem;">
            <a href="/homepage/">← Back to GoLinks</a> | 
            <a href="/docs/">All Documents</a>
        </footer>
    </div>
</body>
</html>`

	t, err := template.New("document").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := t.Execute(w, data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
