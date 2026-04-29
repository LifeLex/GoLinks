package docs

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golinks/internal/platform/logger"

	"gopkg.in/yaml.v2"
)

// Service holds the docs use cases — file access goes through the Store port,
// frontmatter parsing lives here in the core.
type Service struct {
	store  Store
	logger *logger.Logger
}

// NewService wires the docs use cases against the storage port.
func NewService(store Store, log *logger.Logger) *Service {
	log.Info("Document service initialized")
	return &Service{store: store, logger: log}
}

// GetDocument reads a document and returns the raw source plus parsed metadata.
// If the filename has no extension, .md is tried first then .mdx.
func (s *Service) GetDocument(ctx context.Context, filename string) (*DocumentSource, error) {
	resolved, content, err := s.readWithExtensionFallback(ctx, filename)
	if err != nil {
		return nil, err
	}

	docType := "markdown"
	if strings.HasSuffix(resolved, ".mdx") {
		docType = "mdx"
	}

	metaData, _ := splitFrontmatter(content)
	info := DocumentInfo{
		Title:       getStringFromMeta(metaData, "title", strings.TrimSuffix(resolved, filepath.Ext(resolved))),
		Description: getStringFromMeta(metaData, "description", ""),
		Type:        docType,
		Path:        resolved,
		Metadata:    metaData,
	}

	return &DocumentSource{
		Source:   string(content),
		Type:     docType,
		Metadata: info,
	}, nil
}

// SaveDocument persists an uploaded document.
func (s *Service) SaveDocument(ctx context.Context, filename string, content io.Reader) error {
	return s.store.Write(ctx, filename, content)
}

// ListDocuments returns metadata for every document, peeking at frontmatter for titles.
func (s *Service) ListDocuments(ctx context.Context) ([]DocumentInfo, error) {
	names, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	docs := make([]DocumentInfo, 0, len(names))
	for _, name := range names {
		if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".mdx") {
			continue
		}

		docType := "markdown"
		if strings.HasSuffix(name, ".mdx") {
			docType = "mdx"
		}

		title := strings.TrimSuffix(name, filepath.Ext(name))
		description := ""
		if data, err := s.store.Read(ctx, name); err == nil {
			if meta, _ := splitFrontmatter(data); meta != nil {
				title = getStringFromMeta(meta, "title", title)
				description = getStringFromMeta(meta, "description", "")
			}
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

// DeleteDocument removes a document.
func (s *Service) DeleteDocument(ctx context.Context, filename string) error {
	return s.store.Delete(ctx, filename)
}

// readWithExtensionFallback reads `filename`. If the user passed no extension,
// it tries `filename.md` then `filename.mdx` so URL paths can be extension-free.
func (s *Service) readWithExtensionFallback(ctx context.Context, filename string) (string, []byte, error) {
	if strings.HasSuffix(filename, ".md") || strings.HasSuffix(filename, ".mdx") {
		content, err := s.store.Read(ctx, filename)
		if err != nil {
			return "", nil, err
		}
		return filename, content, nil
	}

	for _, ext := range []string{".md", ".mdx"} {
		candidate := filename + ext
		if content, err := s.store.Read(ctx, candidate); err == nil {
			return candidate, content, nil
		} else if !errors.Is(err, ErrNotFound) {
			return "", nil, err
		}
	}
	return "", nil, fmt.Errorf("%w: %s", ErrNotFound, filename)
}

// splitFrontmatter separates a leading YAML `---` block from the body. Returns
// (nil, original) if no frontmatter is present.
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

func getStringFromMeta(meta map[string]interface{}, key, defaultValue string) string {
	if value, ok := meta[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}
