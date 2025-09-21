package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// DocumentService handles document rendering and management
type DocumentService struct {
	docsPath string
	markdown goldmark.Markdown
}

// DocumentInfo contains metadata about a document
type DocumentInfo struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // "markdown" or "mdx"
	Path        string                 `json:"path"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RenderResult contains the rendered document and metadata
type RenderResult struct {
	HTML     string       `json:"html"`
	Metadata DocumentInfo `json:"metadata"`
}

// NewDocumentService creates a new document service
func NewDocumentService(docsPath string) *DocumentService {
	// Configure Goldmark with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Tables
			extension.Strikethrough, // ~~strikethrough~~
			extension.TaskList,      // - [x] task lists
			meta.Meta,               // Frontmatter support
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Convert line breaks to <br>
			html.WithXHTML(),     // XHTML-compliant output
			html.WithUnsafe(),    // Allow raw HTML (be careful!)
		),
	)

	return &DocumentService{
		docsPath: docsPath,
		markdown: md,
	}
}

// GetDocument retrieves and renders a document by filename
func (s *DocumentService) GetDocument(ctx context.Context, filename string) (*RenderResult, error) {
	// Sanitize filename to prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("document not found: %s", filename)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	// Determine document type
	docType := "markdown"
	if strings.HasSuffix(filename, ".mdx") {
		docType = "mdx"
	}

	// Render based on type
	switch docType {
	case "mdx":
		return s.renderMDX(ctx, filename, content)
	default:
		return s.renderMarkdown(ctx, filename, content)
	}
}

// renderMarkdown renders a markdown document
func (s *DocumentService) renderMarkdown(ctx context.Context, filename string, content []byte) (*RenderResult, error) {
	var buf bytes.Buffer
	context := parser.NewContext()

	// Parse and render
	if err := s.markdown.Convert(content, &buf, parser.WithContext(context)); err != nil {
		return nil, fmt.Errorf("failed to render markdown: %w", err)
	}

	// Extract metadata from frontmatter
	metaData := meta.Get(context)
	if metaData == nil {
		metaData = make(map[string]interface{})
	}

	// Create document info
	docInfo := DocumentInfo{
		Title:       getStringFromMeta(metaData, "title", strings.TrimSuffix(filename, filepath.Ext(filename))),
		Description: getStringFromMeta(metaData, "description", ""),
		Type:        "markdown",
		Path:        filename,
		Metadata:    metaData,
	}

	return &RenderResult{
		HTML:     buf.String(),
		Metadata: docInfo,
	}, nil
}

// renderMDX renders an MDX document (simplified for now)
func (s *DocumentService) renderMDX(ctx context.Context, filename string, content []byte) (*RenderResult, error) {
	// For now, treat MDX as enhanced markdown
	// In the future, we could add proper MDX compilation with esbuild

	// Remove JSX-like syntax for basic rendering
	processedContent := s.preprocessMDX(content)

	var buf bytes.Buffer
	context := parser.NewContext()

	// Parse and render as markdown
	if err := s.markdown.Convert(processedContent, &buf, parser.WithContext(context)); err != nil {
		return nil, fmt.Errorf("failed to render MDX: %w", err)
	}

	// Extract metadata
	metaData := meta.Get(context)
	if metaData == nil {
		metaData = make(map[string]interface{})
	}

	// Create document info
	docInfo := DocumentInfo{
		Title:       getStringFromMeta(metaData, "title", strings.TrimSuffix(filename, filepath.Ext(filename))),
		Description: getStringFromMeta(metaData, "description", ""),
		Type:        "mdx",
		Path:        filename,
		Metadata:    metaData,
	}

	return &RenderResult{
		HTML:     buf.String(),
		Metadata: docInfo,
	}, nil
}

// preprocessMDX does basic preprocessing of MDX content
func (s *DocumentService) preprocessMDX(content []byte) []byte {
	contentStr := string(content)

	// Convert simple JSX-like elements to HTML
	// This is a very basic implementation - in production, use proper MDX compiler
	contentStr = strings.ReplaceAll(contentStr, `<div className="alert alert-info">`, `<div class="alert alert-info">`)
	contentStr = strings.ReplaceAll(contentStr, `className=`, `class=`)

	return []byte(contentStr)
}

// SaveDocument saves a document to the filesystem
func (s *DocumentService) SaveDocument(ctx context.Context, filename string, content io.Reader) error {
	// Sanitize filename
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create document file: %w", err)
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("failed to write document content: %w", err)
	}

	return nil
}

// ListDocuments returns a list of available documents
func (s *DocumentService) ListDocuments(ctx context.Context) ([]DocumentInfo, error) {
	entries, err := os.ReadDir(s.docsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}

	var docs []DocumentInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".mdx") {
			continue
		}

		docType := "markdown"
		if strings.HasSuffix(name, ".mdx") {
			docType = "mdx"
		}

		docs = append(docs, DocumentInfo{
			Title: strings.TrimSuffix(name, filepath.Ext(name)),
			Type:  docType,
			Path:  name,
		})
	}

	return docs, nil
}

// DeleteDocument removes a document from the filesystem
func (s *DocumentService) DeleteDocument(ctx context.Context, filename string) error {
	// Sanitize filename
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Helper function to safely get string values from metadata
func getStringFromMeta(meta map[string]interface{}, key, defaultValue string) string {
	if value, ok := meta[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}
