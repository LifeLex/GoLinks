package docs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"golinks/internal/platform/logger"
)

type memStore struct {
	files map[string][]byte
}

func newMemStore(files map[string]string) *memStore {
	bs := make(map[string][]byte, len(files))
	for k, v := range files {
		bs[k] = []byte(v)
	}
	return &memStore{files: bs}
}

func (m *memStore) Read(_ context.Context, filename string) ([]byte, error) {
	if data, ok := m.files[filename]; ok {
		return data, nil
	}
	return nil, ErrNotFound
}

func (m *memStore) Write(_ context.Context, filename string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	m.files[filename] = data
	return nil
}

func (m *memStore) List(_ context.Context) ([]string, error) {
	names := make([]string, 0, len(m.files))
	for k := range m.files {
		names = append(names, k)
	}
	return names, nil
}

func (m *memStore) Delete(_ context.Context, filename string) error {
	if _, ok := m.files[filename]; !ok {
		return ErrNotFound
	}
	delete(m.files, filename)
	return nil
}

func (m *memStore) Exists(_ context.Context, filename string) (bool, error) {
	_, ok := m.files[filename]
	return ok, nil
}

func TestService_GetDocument_PicksTypeFromExtension(t *testing.T) {
	store := newMemStore(map[string]string{
		"sample.md":  "# md",
		"sample.mdx": "# mdx",
	})
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	md, err := svc.GetDocument(context.Background(), "sample.md")
	if err != nil {
		t.Fatalf("GetDocument(.md): %v", err)
	}
	if md.Type != "markdown" {
		t.Errorf("type = %q, want markdown", md.Type)
	}

	mdx, err := svc.GetDocument(context.Background(), "sample.mdx")
	if err != nil {
		t.Fatalf("GetDocument(.mdx): %v", err)
	}
	if mdx.Type != "mdx" {
		t.Errorf("type = %q, want mdx", mdx.Type)
	}
}

func TestService_GetDocument_ExtensionFallback(t *testing.T) {
	store := newMemStore(map[string]string{"sample.md": "# md"})
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	doc, err := svc.GetDocument(context.Background(), "sample")
	if err != nil {
		t.Fatalf("GetDocument(no ext): %v", err)
	}
	if doc.Metadata.Path != "sample.md" {
		t.Errorf("path = %q, want sample.md", doc.Metadata.Path)
	}
}

func TestService_GetDocument_FrontmatterParsed(t *testing.T) {
	src := "---\ntitle: Hello\ndescription: World\n---\n# body\n"
	store := newMemStore(map[string]string{"a.md": src})
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	doc, err := svc.GetDocument(context.Background(), "a.md")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Metadata.Title != "Hello" {
		t.Errorf("title = %q, want Hello", doc.Metadata.Title)
	}
	if doc.Metadata.Description != "World" {
		t.Errorf("description = %q, want World", doc.Metadata.Description)
	}
	if !strings.Contains(doc.Source, "# body") {
		t.Error("source should retain body")
	}
}

func TestService_GetDocument_NotFound(t *testing.T) {
	store := newMemStore(nil)
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	_, err := svc.GetDocument(context.Background(), "missing.md")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestService_SaveAndDelete(t *testing.T) {
	store := newMemStore(nil)
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	if err := svc.SaveDocument(context.Background(), "new.md", bytes.NewBufferString("# new")); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}
	if ok, _ := store.Exists(context.Background(), "new.md"); !ok {
		t.Error("expected new.md to exist after save")
	}

	if err := svc.DeleteDocument(context.Background(), "new.md"); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
	if ok, _ := store.Exists(context.Background(), "new.md"); ok {
		t.Error("expected new.md to be gone after delete")
	}
}

func TestService_ListDocuments_FiltersByExtension(t *testing.T) {
	store := newMemStore(map[string]string{
		"a.md":     "x",
		"b.mdx":    "x",
		"junk.txt": "ignored",
	})
	svc := NewService(store, logger.New(logger.Config{Level: "info"}))

	docs, err := svc.ListDocuments(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Errorf("want 2 docs, got %d", len(docs))
	}
}
