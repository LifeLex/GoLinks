// Package filesystem implements outbound storage adapters backed by the local
// filesystem. Today: the docs/ folder. The package depends only on core/docs
// (for the Store port and ErrNotFound).
package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golinks/internal/core/docs"
)

// DocStore implements docs.Store against a local directory.
type DocStore struct {
	root string
}

// NewDocStore creates a docs.Store that reads/writes to the given directory.
// The directory must already exist; the store does not create it.
func NewDocStore(root string) *DocStore {
	return &DocStore{root: root}
}

// Read returns a document's bytes. Maps os.IsNotExist to docs.ErrNotFound.
func (s *DocStore) Read(_ context.Context, filename string) ([]byte, error) {
	path := s.path(filename)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, docs.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read document %q: %w", filename, err)
	}
	return content, nil
}

// Write creates or overwrites a document.
func (s *DocStore) Write(_ context.Context, filename string, content io.Reader) error {
	path := s.path(filename)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create document file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("failed to write document content: %w", err)
	}
	return nil
}

// List returns every regular file in the docs root.
func (s *DocStore) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("failed to read docs directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

// Delete removes a document.
func (s *DocStore) Delete(_ context.Context, filename string) error {
	if err := os.Remove(s.path(filename)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return docs.ErrNotFound
		}
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// Exists reports whether a document is present without reading it.
func (s *DocStore) Exists(_ context.Context, filename string) (bool, error) {
	_, err := os.Stat(s.path(filename))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// path sanitises filename via filepath.Base and joins it under the root,
// preventing path traversal regardless of caller hygiene.
func (s *DocStore) path(filename string) string {
	return filepath.Join(s.root, filepath.Base(filename))
}
