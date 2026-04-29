package docs

import "errors"

// ErrNotFound is returned by Store implementations when a document is missing.
// Callers unwrap with errors.Is(err, ErrNotFound).
var ErrNotFound = errors.New("document not found")
