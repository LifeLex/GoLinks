# Architecture

This document describes how GoLinks is wired end-to-end: the hexagonal Go backend, the React/Vite SPA, the way they're glued together into a single binary, and exactly what each HTTP endpoint does.

`README.md` covers what the app is and how to use it. `CLAUDE.md` captures conventions for contributors. This file is the implementation reference.

---

## At a glance

```
┌────────────────────────────────────────────────────────────────────┐
│                         golinks (single binary)                    │
│                                                                    │
│  cmd/server/main.go              composition root                  │
│    └── gorilla/mux router                                          │
│         ├── /query/{path:.*}            → 302 redirect             │
│         ├── /api/links       (GET/POST) → JSON                     │
│         ├── /update/         (POST)     → JSON (legacy form)       │
│         ├── /api/docs        (GET/POST) → JSON                     │
│         ├── /api/docs/{file} (GET/DEL)  → JSON                     │
│         └── /*  (catch-all)             → embedded SPA / index.html│
│                                                                    │
│  internal/                                                         │
│    core/<feature>/  (entities, ports, use cases)                   │
│    adapters/<type>/ (httpapi, persistence, filesystem)             │
│    platform/        (config, logger, database{connect,migrate})    │
│                                                                    │
│  Database: GORM → glebarez/sqlite (default) | gorm.io/postgres     │
│  Migrations: Goose (per-dialect SQL files, embedded)               │
│                                                                    │
│  web/frontend/                                                     │
│    React 18 + TS + Vite + Tailwind + shadcn/ui                     │
│    TanStack Query · react-hook-form+zod · @mdx-js/mdx              │
│    └── dist/  ◀── //go:embed all:dist  (web/frontend/embed.go)     │
│                                                                    │
│  docs/        ── on-disk markdown/MDX (read at runtime)            │
│  data/        ── SQLite database file (when DRIVER=sqlite)         │
└────────────────────────────────────────────────────────────────────┘
```

One process. Two TCP listeners only in development (`:8080` Go, `:5173` Vite proxying back). In production: one listener, `:8080`. Pure-Go build — `CGO_ENABLED=0` everywhere.

---

## Hexagonal layout

```
cmd/server/main.go                     entrypoint + DI wiring
internal/
├── core/                              business logic — depends only on its own ports
│   ├── links/
│   │   ├── entity.go                  Shortcut, KeywordInfo, PopularQuery, LinkRequest
│   │   ├── ports.go                   Repository, QueryRepository (interfaces)
│   │   ├── service.go                 Service: GetLink, UpdateLink, GetAllKeywords, GetRecentQueries
│   │   ├── errors.go                  InvalidQueryError
│   │   └── service_test.go            mocks live alongside the test
│   └── docs/
│       ├── entity.go                  DocumentInfo, DocumentSource
│       ├── ports.go                   Store interface
│       ├── service.go                 Service: parses frontmatter, delegates I/O to Store
│       ├── errors.go                  ErrNotFound
│       └── service_test.go
├── adapters/                          implementations of the ports
│   ├── httpapi/                       inbound: HTTP transport
│   │   ├── helpers.go                 writeJSON, userIDFromRequest
│   │   ├── links_handler.go           Redirect, List, Create, UpdateLegacy
│   │   ├── docs_handler.go            Get, List, Upload, Delete
│   │   └── links_handler_test.go      mock LinkService, table-driven tests
│   ├── persistence/                   outbound: GORM repositories
│   │   ├── models.go                  shortcutRow, queryRow, tagRow (unexported) + Models() accessor
│   │   ├── mapper.go                  domain ↔ row conversion
│   │   ├── links_repo.go              implements core/links.Repository
│   │   ├── queries_repo.go            implements core/links.QueryRepository
│   │   ├── repos_test.go              real in-memory SQLite via GORM + Goose
│   │   └── migrations/                Goose Go-based migrations (single folder, dialect-agnostic)
│   │       ├── doc.go                 package documentation + conventions
│   │       └── 00001_initial_schema.go  AutoMigrate via GORM
│   └── filesystem/                    outbound: docs/ folder
│       ├── docs_store.go              implements core/docs.Store
│       └── docs_store_test.go         t.TempDir-based tests
└── platform/                          cross-cutting infrastructure
    ├── config/
    │   ├── config.go                  env loading; DatabaseDriver + DatabaseURL
    │   └── config_test.go
    ├── logger/
    │   └── logger.go                  slog wrapper
    └── database/
        ├── connect.go                 OpenGorm(driver, url) — sqlite or postgres
        ├── migrate.go                 Goose-driven migration runner + WithGorm/GormFromContext helpers
        └── migrate_test.go            external test package (`database_test`) to break import cycle
web/frontend/                          Vite + React SPA + embed.go
docs/                                  user-uploaded .md / .mdx, read from disk at runtime
docker-compose.yml                     GoLinks + Postgres for local dev
.env.example                           every supported env var with examples
```

Frontend embed lives at `web/frontend/embed.go` rather than under `internal/platform/frontend/` because `//go:embed` cannot use relative `..` paths. The embed file must be a sibling of `dist/`. Conceptually it's an inbound adapter; physically it sits next to the JS code so the directive resolves.

---

## Backend layers

### `cmd/server/main.go` — entrypoint

Composition root. Loads config, opens the database, runs migrations, instantiates each adapter, wires services, registers routes, starts the HTTP server with a graceful-shutdown loop.

The full DI graph lives in this one file. No constructor is called anywhere else in the codebase.

### `internal/core/<feature>/` — business logic

Two features today: `links` and `docs`. Each feature folder is a complete mini-hexagon:

- `entity.go` — domain types (no tags except `json` for API responses).
- `ports.go` — interfaces the use cases consume (`Repository`, `QueryRepository`, `Store`).
- `service.go` — use case implementation (`Service` struct).
- `errors.go` — package-level sentinel errors and typed errors (`InvalidQueryError`, `ErrNotFound`).
- `service_test.go` — table-driven tests with hand-written mocks.

The core never imports an adapter. Adapters import core to satisfy its ports.

### `internal/adapters/<type>/` — implementations

- **`httpapi/`** — inbound HTTP. Each handler exposes a `Register(*mux.Router)` method that wires its routes. Package name is `httpapi` rather than `http` to avoid shadowing the standard library.
- **`persistence/`** — GORM repositories. One implementation per port; GORM dispatches on the connection's dialect at startup.
- **`filesystem/`** — local-disk storage for the docs/ folder. Implements the `docs.Store` port and surfaces `docs.ErrNotFound` when files are missing.

### `internal/platform/` — cross-cutting infrastructure

- **`config/`** — `godotenv` + env-var loading. `Config` carries `DatabaseDriver`, `DatabaseURL`, the legacy `DatabasePath` fallback, and the rest.
- **`logger/`** — slog wrapper exposing Printf-style `Info`/`Warn`/`Error`/`Debug` helpers and a process-wide default logger.
- **`database/`** — `OpenGorm(driver, url)` returns a `*gorm.DB` for the requested backend. `Migrate(db, driver)` runs Goose against the right per-dialect migration directory. The `migrations/` subdirectory ships embedded into the binary via `//go:embed`.

### Database layer (GORM + Goose)

- `OpenGorm` dispatches on the `Driver` value: `DriverSQLite` opens `glebarez/sqlite` (pure Go), `DriverPostgres` opens `gorm.io/driver/postgres` (uses `pgx`, also pure Go).
- For SQLite, `OpenGorm` injects `?_pragma=foreign_keys(1)` if the URL doesn't already specify one, so FK enforcement matches Postgres behaviour out of the box.
- `Migrate` extracts the underlying `*sql.DB` from GORM, attaches the `*gorm.DB` to a context via `WithGorm`, and calls `goose.UpContext`. Goose dialect string is mapped from our `Driver` (SQLite uses `"sqlite3"` in Goose terminology).
- **Migrations are Go code, not SQL.** They live at `internal/adapters/persistence/migrations/`, in a single folder regardless of dialect. Each migration registers itself via `init()` and delegates schema work to GORM (`AutoMigrate`, `Migrator()`). GORM translates to dialect-specific SQL — `INTEGER PRIMARY KEY AUTOINCREMENT` vs `BIGSERIAL`, `DATETIME` vs `TIMESTAMP`, reserved-word quoting — without the migration author having to think about it.
- The composition root blank-imports the migrations package (`_ "golinks/internal/adapters/persistence/migrations"`) so `init()` runs at startup. Tests do the same.
- Goose's `goose_db_version` table is created automatically at first run; subsequent boots are no-ops unless new migrations land.
- For genuinely dialect-specific work (FTS5 in SQLite, `tsvector` in Postgres) a single migration may dispatch on the dialect via raw `db.Exec`. Stays in the same folder — the dispatch lives inside the Go function, not the filesystem.

### `web/frontend/embed.go` — SPA bridge

Compile-time `//go:embed all:dist` pulls the Vite build output into the binary. The exported `Handler(reservedPrefixes ...)` returns an `http.Handler` that:

- Refuses non-GET/HEAD with 405.
- Refuses any path starting with `api/` or `query/` (defensive backstop).
- Serves real files from the embedded FS when they exist (`/assets/*`, `/favicon.ico`).
- Falls back to `index.html` with `Cache-Control: no-cache` for everything else, so React Router can take over on hard refreshes.

If the embedded `dist/` is empty (e.g. fresh clone without `npm run build`), the handler returns 503 with a helpful message — combined with the committed stub `index.html`, this guarantees the binary is always runnable.

---

## Frontend architecture

(Unchanged from the prior version — see `CLAUDE.md` for the style rules. A short summary follows.)

- **`src/main.tsx`** mounts `<App />` inside `QueryClientProvider`, `BrowserRouter`, and a sonner `<Toaster>`.
- **`src/App.tsx`** is the route table: `/`, `/setup`, `/docs`, `/docs/:filename`, plus a custom 404.
- **State:** TanStack Query for server state; `useSearchParams` for URL state; `react-hook-form` + `zod` for forms; `useState` for purely local UI state.
- **API client:** `src/lib/api.ts` exports a typed `api` object. Errors throw `ApiError`; toasts surface them via sonner.
- **MDX:** `src/lib/mdx.tsx` runs `@mdx-js/mdx evaluate()` in the browser with an explicit `mdxComponents` map (Alert, Card, Tabs, Button, plus custom table primitives for GFM tables).

---

## Build & distribution

### Development

`make dev` runs the Go server (via `air` if installed) and the Vite dev server (`:5173`) concurrently. `vite.config.ts` proxies `/api/*` and `/query/*` to `http://localhost:8080`.

For breakpoint debugging, see `.zed/debug.json` (Backend Go via Delve, Vite via Node, Chrome with source maps).

### Production single binary

`make build` runs:

1. `npm ci && npm run build` inside `web/frontend/` → produces `web/frontend/dist/`.
2. `CGO_ENABLED=0 go build -o build/golinks ./cmd/server` — `//go:embed all:dist` pulls dist into the binary; Goose migrations are also embedded.

Result: a single ~20 MB pure-Go binary that needs only the `docs/` directory and a database (SQLite file or Postgres connection) at runtime.

### Docker

Three-stage `Dockerfile`:

1. `node:20-alpine` builds the SPA.
2. `golang:1.21-alpine` builds the binary with `CGO_ENABLED=0` — no `gcc`, no `sqlite-dev`, no C toolchain.
3. `alpine:3.18` runtime with `ca-certificates` + `tzdata` only. Copies the binary and the `docs/` directory. Drops to a non-root `golinks` user. Exposes 8080.

### `docker-compose.yml`

Local dev stack with Postgres. The `app` service runs GoLinks with `DATABASE_DRIVER=postgres`; the `db` service is `postgres:16-alpine` with a named volume. Boot with `docker compose up --build`.

---

## Endpoint reference

Routes are registered in `internal/adapters/httpapi/links_handler.go:Register` (links + redirect + legacy form) and `internal/adapters/httpapi/docs_handler.go:Register` (docs CRUD), with the SPA catch-all wired in `cmd/server/main.go`. gorilla/mux matches in registration order.

### `GET /query/{path:.*}` — golink redirect

Server-side 302. The contract is browser-search-engine compatible.

**Flow** (`LinksHandler.Redirect`):

1. Strip trailing slash from the captured `path`.
2. Call `linkService.GetLink(ctx, path, "")`.
3. On success → `302` to the resolved URL.
4. On `links.InvalidQueryError` → `302` to `${BASE_URL}/?missing=<path>` (the SPA shows a toast).
5. On any other error → `500`.

**Resolution semantics** (`core/links.Service.GetLink`):

- Look up `word` in the repository. If found:
  - Log a hit (best-effort; logging failure does not fail the request).
  - If the stored `link` is itself a keyword (not `http(s)://...`), recurse — enables aliases.
  - If the stored `link` contains `{*}`, substitute the `searchTerm` (URL-encoded).
  - Return the URL.
- If not found *and* `word` contains spaces, peel the last token off and treat it as a search term (`go google cats` → look up `google cats`, fail, then `google` with searchTerm `cats`).
- If still not found → `links.InvalidQueryError`.

### `GET /api/links` — list keywords + recent queries

`LinksHandler.List`. One round-trip per homepage render.

```json
{
  "keywords":      [ { "word": "...", "link": "...", "created_at": "..." }, ... ],
  "recent_queries":[ { "count": 5, "word": "...", "link": "..." }, ... ],
  "base_url":      "http://localhost:8080"
}
```

- `keywords` ← `links.Service.GetAllKeywords()` → all rows from `linktable`, deduped to the latest version of each word, filtered to URL-shaped links.
- `recent_queries` ← `links.Service.GetRecentQueries()` → top 20 by count over the last 3 days, joined back to `linktable` for the URL.
- `base_url` ← `cfg.BaseURL`, used by the SPA to display the search-engine URL.

### `POST /api/links` — create or update

`LinksHandler.Create`. JSON body: `{"word":"...","link":"..."}`.

1. Decode JSON; trim whitespace on both fields.
2. `links.Service.UpdateLink` validates: non-empty `word` and `link`; `word` doesn't end in `/`; `link` starts with `http://` or `https://`; `link != word`.
3. Insert via the GORM repository.
4. Return `{"success": true}` (200) or `400` with the validation message.

### `POST /update/` — legacy form-encoded

`LinksHandler.UpdateLegacy`. Same semantics as `POST /api/links`, but accepts `application/x-www-form-urlencoded` and returns plain text. Kept for browsers configured against the pre-migration `/update/` endpoint.

### `GET /api/docs` — list documents

`DocsHandler.List`. Returns one entry per `.md` / `.mdx` file in `docs/`, with frontmatter title/description peeked from each. Delegates to `core/docs.Service.ListDocuments`, which calls `Store.List` then `Store.Read` for each file to extract metadata.

### `GET /api/docs/{filename}` — fetch raw source

`DocsHandler.Get`. Returns the full file contents (frontmatter included) plus parsed metadata.

```json
{
  "source":   "---\ntitle: ...\n---\n# ...",
  "type":     "markdown|mdx",
  "metadata": { "title": "...", "description": "...", "type": "...", "path": "...", "metadata": { ... } }
}
```

Extension fallback: if the URL omits `.md`/`.mdx`, the service tries `.md` first then `.mdx`. 404 (mapped from `docs.ErrNotFound`) if neither exists.

### `POST /api/docs` — upload

`DocsHandler.Upload`. `multipart/form-data` with field `file`.

1. Parse multipart form (10 MB limit).
2. Reject filenames not ending in `.md` or `.mdx`.
3. Delegate to `core/docs.Service.SaveDocument` → `filesystem.DocStore.Write` (which sanitises via `filepath.Base`).
4. Return `{success, filename, message, url}`.

⚠️ **No authentication.** With runtime MDX compilation, an unauthenticated upload is effectively stored-XSS. Gate behind auth or restrict uploads to `.md` before public deployment.

### `DELETE /api/docs/{filename}` — delete

`DocsHandler.Delete`. `filesystem.DocStore.Delete` returns `docs.ErrNotFound` for missing files (mapped to 404).

### `GET /` and unknown routes — embedded SPA

The catch-all hands the request to `frontend.Handler("api", "query")`. Behaviour described under "web/frontend/embed.go" above.

---

## Configuration

Loaded by `internal/platform/config/config.go`. All env vars optional; defaults shown.

| Variable          | Default                  | Used by                                                                |
| ----------------- | ------------------------ | ---------------------------------------------------------------------- |
| `PORT`            | `8080`                   | HTTP server bind                                                       |
| `BASE_URL`        | `http://localhost:8080`  | Returned in `/api/links`; used in 302 fallback target                  |
| `ENVIRONMENT`     | `development`            | Logged on startup; reserved for future env-aware logic                 |
| `LOG_LEVEL`       | `info`                   | `logger.Config.Level`                                                  |
| `DATABASE_DRIVER` | `sqlite`                 | `database.OpenGorm` dispatch (`sqlite` or `postgres`)                  |
| `DATABASE_URL`    | derived from `DATABASE_PATH` when empty and driver=sqlite | Connection string passed straight to GORM |
| `DATABASE_PATH`   | `golinks.db`             | Deprecated. Used to synthesise `DATABASE_URL` when it's empty (sqlite) |

`.env` at the repo root is auto-loaded if present. See `.env.example` for the canonical template.

---

## Database schema

Created and managed by Goose. Migrations live in a single folder at `internal/adapters/persistence/migrations/` and are written as Go functions that delegate schema work to GORM — one set of code, two backends. Dialect-specific SQL (autoincrement vs. `BIGSERIAL`, `DATETIME` vs. `TIMESTAMP WITH TIME ZONE`, reserved-word quoting) is generated by GORM's dialector at apply time.

| Table         | Columns                                                          | Notes                                          |
| ------------- | ---------------------------------------------------------------- | ---------------------------------------------- |
| `linktable`   | `id`, `word`, `link`, `user`, `created_at`                       | Multiple rows per word allowed; latest wins.   |
| `queries`     | `query_id`, `word_id` → `linktable.id`, `created_at`             | Hit log.                                       |
| `tags`        | `id`, `word_id` → `linktable.id`, `tag`                          | Reserved; not yet used by the app.             |
| `goose_db_version` | (Goose-managed)                                              | Migration bookkeeping.                         |

Foreign keys are declared with `ON DELETE RESTRICT ON UPDATE RESTRICT`. SQLite enforces them because the connection enables `_pragma=foreign_keys(1)`.

Indexes:
- `idx_linktable_word` on `linktable.word`
- `idx_queries_word_id` on `queries.word_id`
- `idx_queries_created_at` on `queries.created_at`

---

## Security & known TODOs

- **`POST /api/docs` is unauthenticated.** Combined with runtime MDX compilation, this is the highest-risk gap. Mitigations: gate behind a token, require auth, or restrict uploads to `.md`.
- **No CSRF protection** on `POST /api/links` or `/update/`. Acceptable for a single-user tool on localhost; revisit if exposed publicly.
- **`getUserID` returns `"DefaultUser"`** unconditionally (`internal/adapters/httpapi/helpers.go`). Real auth never landed; the `user` column is a placeholder.

## Future / aspirational

OpenTelemetry, distributed rate-limiting, retry/backoff, circuit breakers — none are wired today and none should be added speculatively. The current scope is a single-instance tool with one external dependency at most (Postgres, when chosen).
