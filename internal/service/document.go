package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golinks/internal/logger"

	"gopkg.in/yaml.v2"
)

// DocumentService stores and retrieves markdown/MDX documents on disk.
//
// Compared to the previous incarnation this service does NOT render the
// documents server-side: the Vite/React frontend handles MDX compilation at
// runtime via @mdx-js/mdx. The Go side is limited to file I/O plus lightweight
// frontmatter parsing so the client knows the title/description without
// having to parse the whole document twice.
type DocumentService struct {
	docsPath string
	logger   *logger.Logger
}

// DocumentInfo contains metadata about a document.
type DocumentInfo struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // "markdown" or "mdx"
	Path        string                 `json:"path"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentSource is the raw file contents plus its parsed metadata.
// The client receives this untouched and compiles MDX in the browser.
type DocumentSource struct {
	Source   string       `json:"source"`
	Type     string       `json:"type"`
	Metadata DocumentInfo `json:"metadata"`
}

// NewDocumentService creates a new document service rooted at docsPath.
func NewDocumentService(docsPath string, log *logger.Logger) *DocumentService {
	log.Info("Initializing document service: %s", docsPath)
	return &DocumentService{
		docsPath: docsPath,
		logger:   log,
	}
}

// GetDocument reads a document by filename and returns its raw source plus metadata.
func (s *DocumentService) GetDocument(ctx context.Context, filename string) (*DocumentSource, error) {
	_ = ctx
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("document not found: %s", filename)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document: %w", err)
	}

	docType := "markdown"
	if strings.HasSuffix(filename, ".mdx") {
		docType = "mdx"
	}

	metaData, body := splitFrontmatter(content)
	info := DocumentInfo{
		Title:       getStringFromMeta(metaData, "title", strings.TrimSuffix(filename, filepath.Ext(filename))),
		Description: getStringFromMeta(metaData, "description", ""),
		Type:        docType,
		Path:        filename,
		Metadata:    metaData,
	}

	// Hand back the full source (including frontmatter) so the client can
	// decide whether to strip it itself. remark/MDX pipelines tolerate both.
	_ = body
	return &DocumentSource{
		Source:   string(content),
		Type:     docType,
		Metadata: info,
	}, nil
}

// SaveDocument writes a document to disk, creating or overwriting.
func (s *DocumentService) SaveDocument(ctx context.Context, filename string, content io.Reader) error {
	_ = ctx
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create document file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("failed to write document content: %w", err)
	}
	return nil
}

// ListDocuments returns metadata for every .md / .mdx file in the docs folder.
func (s *DocumentService) ListDocuments(ctx context.Context) ([]DocumentInfo, error) {
	_ = ctx
	entries, err := os.ReadDir(s.docsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}

	docs := make([]DocumentInfo, 0, len(entries))
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

		// Cheap frontmatter peek for the list view: read only if the first
		// line is `---`, otherwise fall back to the filename.
		title := strings.TrimSuffix(name, filepath.Ext(name))
		description := ""
		if meta := peekFrontmatter(filepath.Join(s.docsPath, name)); meta != nil {
			title = getStringFromMeta(meta, "title", title)
			description = getStringFromMeta(meta, "description", "")
		}

		docs = append(docs, DocumentInfo{
			Title:       title,
			Description: description,
			Type:        docType,
			Path:        name,
		})
	}

	return docs, nil
}

// DeleteDocument removes a document from disk.
func (s *DocumentService) DeleteDocument(ctx context.Context, filename string) error {
	_ = ctx
	filename = filepath.Base(filename)
	filePath := filepath.Join(s.docsPath, filename)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// splitFrontmatter separates a leading YAML `---` block from the body.
// Returns (nil, original) if no frontmatter is present.
func splitFrontmatter(content []byte) (map[string]interface{}, []byte) {
	const delim = "---"
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 1<<16), 1<<20)

	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != delim {
		return nil, content
	}

	var yamlBuf bytes.Buffer
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == delim {
			closed = true
			break
		}
		yamlBuf.WriteString(line)
		yamlBuf.WriteByte('\n')
	}
	if !closed {
		return nil, content
	}

	var meta map[string]interface{}
	if err := yaml.Unmarshal(yamlBuf.Bytes(), &meta); err != nil {
		return nil, content
	}

	var bodyBuf bytes.Buffer
	for scanner.Scan() {
		bodyBuf.WriteString(scanner.Text())
		bodyBuf.WriteByte('\n')
	}
	return meta, bodyBuf.Bytes()
}

func peekFrontmatter(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	meta, _ := splitFrontmatter(data)
	return meta
}

func getStringFromMeta(meta map[string]interface{}, key, defaultValue string) string {
	if value, ok := meta[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}
