package docs

import (
	"context"
	"io"
)

// Store is the outbound port for raw document storage. It hides the choice of
// backend (local filesystem today; possibly object storage later) from the
// service layer. All file paths are relative — implementations are responsible
// for sanitising and resolving them.
type Store interface {
	// Read returns the raw bytes of a document. Returns ErrNotFound if missing.
	Read(ctx context.Context, filename string) ([]byte, error)
	// Write persists a document, overwriting any existing file with the same name.
	Write(ctx context.Context, filename string, content io.Reader) error
	// List enumerates every document filename in the store.
	List(ctx context.Context) ([]string, error)
	// Delete removes a document by filename.
	Delete(ctx context.Context, filename string) error
	// Exists reports whether a document with this filename is present.
	Exists(ctx context.Context, filename string) (bool, error)
}
