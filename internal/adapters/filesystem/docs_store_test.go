package filesystem

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golinks/internal/core/docs"
)

func TestDocStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewDocStore(dir)
	ctx := context.Background()

	if err := store.Write(ctx, "hello.md", bytes.NewBufferString("# hi")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := store.Read(ctx, "hello.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "# hi" {
		t.Errorf("Read returned %q, want %q", got, "# hi")
	}

	exists, err := store.Exists(ctx, "hello.md")
	if err != nil || !exists {
		t.Errorf("Exists = (%v, %v), want (true, nil)", exists, err)
	}

	if err := store.Delete(ctx, "hello.md"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "hello.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should be gone after Delete, stat err = %v", err)
	}
}

func TestDocStore_ReadMissing(t *testing.T) {
	store := NewDocStore(t.TempDir())
	_, err := store.Read(context.Background(), "missing.md")
	if !errors.Is(err, docs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestDocStore_DeleteMissing(t *testing.T) {
	store := NewDocStore(t.TempDir())
	err := store.Delete(context.Background(), "missing.md")
	if !errors.Is(err, docs.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestDocStore_List_FiltersDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewDocStore(dir)
	names, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "a.md" {
		t.Errorf("got %v, want [a.md]", names)
	}
}

func TestDocStore_PathTraversalSanitised(t *testing.T) {
	dir := t.TempDir()
	store := NewDocStore(dir)
	ctx := context.Background()

	// Attempt to write outside the root via "..". filepath.Base should strip the prefix.
	if err := store.Write(ctx, "../escape.md", bytes.NewBufferString("nope")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// File should land at <dir>/escape.md, not at the parent.
	if _, err := os.Stat(filepath.Join(dir, "escape.md")); err != nil {
		t.Errorf("expected sanitised file at root, got %v", err)
	}
	parent := filepath.Dir(dir)
	if _, err := os.Stat(filepath.Join(parent, "escape.md")); err == nil {
		t.Error("path traversal succeeded — file ended up in parent dir")
	}
}
