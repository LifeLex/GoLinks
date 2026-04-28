// Package frontend embeds the built Vite/React SPA and serves it.
//
// The go:embed directive below pulls every file under web/frontend/dist/ into
// the binary at compile time. For SPA routing to work we serve concrete files
// when they exist and fall back to index.html for every other GET — so a hard
// refresh on /docs/sample.md still lands on the React router.
//
// If the dist directory is empty (e.g. `go build` was run without `npm run
// build` first), Handler degrades to a helpful 503 rather than a crash so the
// caller can diagnose the missing build step.
package frontend

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// Requests starting with any of `reservedPrefixes` are rejected with 404 so
// they can be handled by other routes; in practice we only call Handler on
// the catch-all route, so the reservation is belt-and-braces.
func Handler(reservedPrefixes ...string) http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return brokenHandler(fmt.Errorf("failed to locate embedded frontend: %w", err))
	}

	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return brokenHandler(fmt.Errorf(
			"embedded frontend is empty — run `npm run build` in web/frontend/ before `go build`: %w",
			err,
		))
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		reqPath := strings.TrimPrefix(r.URL.Path, "/")
		for _, p := range reservedPrefixes {
			if strings.HasPrefix(reqPath, strings.TrimPrefix(p, "/")) {
				http.NotFound(w, r)
				return
			}
		}

		if reqPath != "" {
			if f, err := sub.Open(reqPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		serveIndex(w, sub)
	})
}

func serveIndex(w http.ResponseWriter, sub fs.FS) {
	f, err := sub.Open("index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = io.Copy(w, f)
}

func brokenHandler(err error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	})
}
