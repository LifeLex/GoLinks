# CLAUDE.md

## Project

**GoLinks** — a minimalist URL shortener inspired by Google's internal golinks system.

**Stack**
- **Backend:** Go (no CGO), `gorilla/mux` for routing, GORM for persistence, Goose for versioned migrations. SQLite (default) via `github.com/glebarez/sqlite` and Postgres via `gorm.io/driver/postgres`. Hexagonal architecture under `internal/`.
- **Frontend:** React 18 + TypeScript + Vite, Tailwind with shadcn/ui primitives, TanStack Query, react-hook-form + zod, react-router-dom, `@mdx-js/mdx` for runtime MDX compilation.
- **Distribution:** Single Go binary. The Vite build output (`web/frontend/dist/`) is embedded via `//go:embed all:dist` in `web/frontend/embed.go`. Goose migrations are embedded too. No `web/` directory at runtime; SQLite or Postgres backend selected at startup via env config.

See `README.md` for user-facing details and `ARCHITECTURE.md` for the implementation reference.

## Repository layout

```
cmd/server/                          Composition root: wires every adapter and use case
internal/
├── core/                            Business logic
│   ├── links/                       golink feature: entity, ports, service
│   └── docs/                        docs feature: entity, ports, service
├── adapters/                        Implementations of the ports
│   ├── httpapi/                     Inbound: HTTP handlers (JSON + redirect)
│   ├── persistence/                 Outbound: GORM-based repositories
│   └── filesystem/                  Outbound: docs/ folder adapter
└── platform/                        Cross-cutting infrastructure
    ├── config/                      Env loading (DATABASE_DRIVER, DATABASE_URL, etc.)
    ├── logger/                      slog wrapper
    └── database/
        ├── connect.go               OpenGorm(driver, url) — sqlite or postgres
        └── migrate.go               Goose-driven migration runner (Go migrations live with persistence)
web/frontend/                        Vite/React SPA + embed.go (frontend embed lives here for go:embed reasons)
docs/                                User-uploaded markdown/MDX (read at runtime)
```

## Backend conventions (Go)

### Architecture
- **Hexagonal:** core depends only on its own ports; adapters depend on core; nothing depends on adapters except `cmd/server/main.go` (the composition root).
- **Per-feature ports:** repository interfaces live next to the use case that consumes them (`core/links/ports.go`, `core/docs/ports.go`). No central `ports/` package — that's a Go anti-pattern (Cox/Cheney).
- **One repository implementation per port:** GORM picks the dialect from the connection. SQLite and Postgres go through the same repository code.
- **Composition root:** every constructor call lives in `cmd/server/main.go`. Don't wire dependencies anywhere else.

### Code style
- Wrap errors with `fmt.Errorf("context: %w", err)`. Use `errors.Is` / `errors.As` to inspect.
- Pass `context.Context` through every layer; service methods take `ctx` as the first arg.
- Defer closing every resource (rows, files, response bodies, DBs).
- Validate input at request boundaries; trust types inside the boundary.
- No global state — everything wired via constructors.

### Persistence (GORM)
- Models are unexported (`shortcutRow`, `queryRow`, `tagRow`) in `adapters/persistence/models.go`. Mapper functions translate between models and domain entities so GORM tags never leak into `core/`.
- Foreign keys are declared via relationship pointer fields with `gorm:"foreignKey:…;references:…;constraint:…"`. AutoMigrate creates them; Goose migrations declare them explicitly in raw SQL.
- For non-trivial queries (GROUP BY + JOIN + aggregations), use a projection struct and `Scan(&rows)` instead of populating the model. Keeps query logic explicit.
- Compute "time windows" in Go (`time.Now().AddDate(...)`) and pass as parameters — never use dialect-specific date functions like SQLite's `datetime('now', '-N days')`.

### Migrations (Goose, single folder)
- All migrations live under `internal/adapters/persistence/migrations/` — one folder, one set of files, regardless of dialect.
- Each migration is a `.go` file (filename `NNNNN_description.go`) that registers itself via `init()` calling `goose.AddMigrationNoTxContext`.
- Migration functions delegate schema work to GORM (`AutoMigrate`, `Migrator()` ops). GORM handles dialect translation, so `INTEGER PRIMARY KEY AUTOINCREMENT` vs `BIGSERIAL` etc. is invisible to the migration author.
- For genuinely dialect-specific work (FTS5 in SQLite, `tsvector` in Postgres) a single migration file may dispatch on the dialect via raw `db.Exec`. Keep the file in this folder; do not split into per-engine subfolders.
- Goose's `goose_db_version` table is created automatically; don't reference it manually.
- The composition root (`cmd/server/main.go`) blank-imports the migrations package so `init()` registrations happen. Tests that run migrations need the same blank import.

### HTTP handlers
- One JSON shape per endpoint, encoded via the `writeJSON(w, status, body)` helper.
- The golink resolver (`/query/{path:.*}`) MUST stay a server-side 302 — it's the contract that lets browser search-engine integrations work.
- `/api/*` is reserved for JSON. The catch-all SPA handler defensively refuses `/api/*` requests.
- Keep `/update/` (form-encoded) as a legacy alias.

### Testing
- Table-driven tests with parallel execution.
- Repository tests open in-memory SQLite via GORM and run the production Goose migrations — the same code path production uses. No fake schema.
- Mock services live next to the consumer: `internal/adapters/httpapi/links_handler_test.go` defines `mockLinkService`; the core tests define mock repositories.
- `go test ./... -race` must pass before merging. CGO must remain off (`CGO_ENABLED=0 go test ./...`).

### Linting & formatting
- `gofmt -s`, `goimports -local golinks`, `golangci-lint run --timeout=3m`. Wired into `make fmt`, `make fix`, `make lint`.

## Frontend conventions

### Architecture
- The SPA owns ALL UI. The Go server returns JSON or 302; it never renders HTML except the embedded `index.html`.
- Components stay presentational. **TanStack Query owns server state** — don't reinvent caching.
- Keep components small and colocated by feature. shadcn primitives live under `components/ui/`; app components one level up.
- No global state library (Redux, Zustand). TanStack Query + URL params + component state cover the surface.

### Styling
- Tailwind utility classes. Only bespoke CSS lives in `src/index.css` (theme tokens + prose overrides).
- Rams-inspired palette ported into shadcn HSL CSS variables; `--primary` is Braun orange. Don't hard-code hex; reference tokens (`bg-primary`, `text-foreground`, `border-border`).
- Long-form rendered docs use `@tailwindcss/typography` with overrides in `index.css`.

### API client
- All HTTP through `src/lib/api.ts`. Never call `fetch` directly from a component.
- Each endpoint gets a typed wrapper. Errors throw `ApiError`; mutation/query `onError` shows them via `sonner` toasts.

### Routing
- `react-router-dom` v6. Routes in `src/App.tsx`. Pages under `src/pages/` never import each other.
- Deep-linkable URLs. State that can be in the URL should be — use `useSearchParams`.

### MDX
- Real client-side compilation via `@mdx-js/mdx evaluate()` in `src/lib/mdx.tsx`. Server returns raw source.
- Components exposed to MDX are explicitly enumerated in `mdxComponents`. No magic auto-discovery.
- **Security:** runtime MDX is JS execution. `POST /api/docs` is unauthenticated — gate it (or restrict uploads to `.md`) before exposing publicly.

### TypeScript
- `tsc -b` must pass. `strict`, `noUnusedLocals`, `noUnusedParameters` are on.
- No `any` unless interfacing with an unyielding library type. Path alias `@/*` maps to `src/*`.

## Workflows

### Development
- `make dev` runs the Go server (with `air` if installed) and the Vite dev server (`:5173`) concurrently. Vite proxies `/api` and `/query` to `:8080`.
- Frontend-only: `make frontend-dev`. Backend-only: `go run ./cmd/server` (the committed stub `dist/index.html` serves a "build the frontend" page until you `make frontend-build`).
- Postgres locally: `docker compose up --build` boots GoLinks against a Postgres container.

### Build
- `make build` runs `npm run build` then `CGO_ENABLED=0 go build`. Pure-Go binary; ~20 MB.
- A stub `dist/index.html` is committed so `git clone && go build` produces a runnable binary even before the frontend has been built.

### Docker
- Three-stage `Dockerfile`: `node:20-alpine` builds the SPA → `golang:1.21-alpine` builds with CGO disabled → `alpine:3.18` runtime with only the binary, `docs/`, and the data volume.
- The runtime image must have no `web/` directory. If you see one, the Dockerfile has regressed.

## Cross-cutting principles

1. **Single artifact.** One Go binary serves API, redirects, and SPA. Don't introduce a separate frontend service.
2. **Trust the boundary.** Validate at request edges. Inside the boundary, types are honest.
3. **Reuse over rewrite.** Before adding a component or helper, check `components/ui/`, `lib/utils.ts`, `lib/api.ts`, the service layer, the persistence helpers.
4. **Boring tech.** Stick to the existing stack unless there's a concrete reason to add a dependency.
5. **Readable, testable, documented.** Every exported Go function gets a GoDoc comment. Every JSON endpoint has at least a smoke test.

## Reversed decisions (recorded explicitly)

- **GORM adopted on 2026-04-28.** Reverses the prior "no ORM" decision (recorded in earlier TODOs as "plain `database/sql` is the explicit convention"). Rationale: SQLite/Postgres dual support without per-dialect query code; explicit FK constraints; less hand-rolled boilerplate. Trade-off accepted: less direct control over generated SQL.
- **Postgres supported on 2026-04-28.** Reverses the prior "single-binary, SQLite-only" decision. Rationale: enables horizontally-scalable deployments and managed-DB integrations. SQLite remains the default for development and single-instance use.

## Aspirational (not yet wired)

- OpenTelemetry tracing/metrics. Adopt when an observability backend is in place.
- Distributed rate-limiting. Single-instance deployment doesn't need it.
- Auth on `POST /api/docs`. Critical before any public deployment because of runtime MDX.
- Retries / circuit breakers / backoff. Add when external dependencies appear.
