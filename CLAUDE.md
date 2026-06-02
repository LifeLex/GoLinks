# CLAUDE.md

## Project

**GoLinks** — a minimalist URL shortener inspired by Google's internal golinks system.

**Stack**
- **Backend:** Go (standard library + `gorilla/mux`), SQLite via `mattn/go-sqlite3`, Clean Architecture under `internal/`.
- **Frontend:** React 18 + TypeScript + Vite, Tailwind CSS with shadcn/ui primitives, TanStack Query for server state, react-hook-form + zod for forms, react-router-dom for client routing, `@mdx-js/mdx` for runtime MDX compilation.
- **Distribution:** Single Go binary. The Vite build output (`web/frontend/dist/`) is embedded via `//go:embed all:dist` in `web/frontend/embed.go` — no `web/` directory exists at runtime.

See `README.md` for user-facing details and commands.

## Repository layout

```
cmd/server/                  Application entrypoint
internal/
├── config/                  Env / dotenv configuration
├── database/                SQLite connection + migrations
├── domain/                  Models (json + db tagged) + sentinel errors
├── handlers/                HTTP handlers (JSON API + redirect) + auth middleware
├── logger/                  Structured logger
├── repository/              Data access layer (shortcuts, queries, users, sessions)
└── service/                 Business logic (links, documents, auth)
web/frontend/                Vite + React SPA
├── src/components/          App components
├── src/components/ui/       shadcn primitives
├── src/pages/               Route-level pages
├── src/lib/                 api.ts, mdx.tsx, utils.ts
├── public/                  Static assets (favicon)
├── dist/                    Build output, embedded into the Go binary
└── embed.go                 go:embed bridge + SPA fallback handler
docs/                        User-uploaded .md / .mdx, read from disk at runtime
```

## Backend conventions (Go)

### Architecture
- Clean Architecture layering: handlers → service → repository → database. Handlers never touch the DB directly.
- Interface-driven: handlers depend on service interfaces declared in the handlers package; services depend on repository interfaces.
- Composition over inheritance; small, purpose-built interfaces.
- No global state — wire dependencies through constructors.

### Code style
- Short, single-purpose functions. Wrap errors with `fmt.Errorf("context: %w", err)`.
- Pass `context.Context` through the call chain; service methods take `ctx` as the first arg.
- Defer closing every resource you open (rows, files, response bodies).
- Validate input at request boundaries — never trust query strings, form values, or JSON bodies.

### HTTP handlers
- One JSON shape per endpoint, encoded via the `writeJSON(w, status, body)` helper. Don't hand-roll JSON in each handler.
- The golink resolver (`/query/{path:.*}`) MUST stay a server-side 302 — it's the contract that lets browser search-engine integrations work. Never replace it with a client-side redirect. It stays **public** unconditionally.
- `/api/*` and `/auth/*` are reserved for JSON. The catch-all SPA handler refuses these prefixes defensively as a backstop against route-registration regressions.
- `/update/` (form-encoded) is a legacy create alias. It now **requires auth** like `POST /api/links` — closing the old unauthenticated write path.

### Authentication & authorization
- **Email + password**, bcrypt-hashed (`golang.org/x/crypto/bcrypt`). Sessions are server-side: an opaque random token lives in an `HttpOnly`, `SameSite=Lax`, `Secure` (prod) cookie, and only its **SHA-256 hash** is stored in the `sessions` table. No JWT, no signing secret.
- **Bootstrap:** the first user created on an empty DB (`POST /auth/setup`) becomes `admin`. Registration is closed afterward; admins create users via `POST /api/users`.
- **Gating model:** a global `Authenticate` middleware loads the optional user into request context (anonymous if no/invalid cookie — it never rejects, so reads stay public). Two subrouters carry guards: `RequireAuth` (401 if anonymous) and `RequireAdmin` (401/403). Wire write routes onto these in `cmd/server/main.go`; never gate inside handler bodies.
- **Reads public, writes authed.** Docs upload/delete are admin-only (runtime MDX evaluates JSX in the browser). Get the current user with `handlers.UserFromContext(ctx)`; it returns nil when anonymous.
- Map auth failures to status codes via the `domain.Err*` sentinels (see `internal/domain/errors.go`) — never leak whether an email exists.

### Testing
- Table-driven unit tests with parallel execution.
- Mock external interfaces with handwritten mocks colocated in `_test.go` files. The `mockLinkService` in `internal/handlers/handler_test.go` is the reference pattern.
- `go test ./... -race` must pass before merging.

### Linting & formatting
- `gofmt -s`, `goimports -local golinks`, `golangci-lint run --timeout=3m`. Wired into `make fmt`, `make fix`, `make lint`.
- `make ci` is the full gate: frontend install + build, lint, test, build.

## Frontend conventions

### Architecture
- The SPA owns ALL UI. The Go server returns JSON or 302; it never renders HTML except the embedded `index.html`.
- Components stay presentational. **TanStack Query owns server state** — don't reinvent caching with `useEffect` + local state for fetched data.
- Keep components small and colocated by feature. shadcn primitives live under `components/ui/`; app components live one level up.
- Don't introduce a global state library (Redux, Zustand). TanStack Query + URL params + component state cover the surface.

### Styling
- **Use Tailwind utility classes.** The only bespoke CSS file is `src/index.css`, which holds theme tokens and prose overrides.
- The palette is a Rams-inspired set ported to shadcn HSL CSS variables. `--primary` is Braun orange — keep it as the distinctive accent. Don't hard-code hex values; reference tokens via `bg-primary`, `text-foreground`, `border-border`, etc.
- Long-form rendered documents use `@tailwindcss/typography` `prose` classes with overrides in `index.css`.
- Border radius flows from `--radius` (4px). Don't introduce arbitrary radius values.

### API client
- All HTTP calls go through `src/lib/api.ts`. Never call `fetch` directly from a component.
- Each endpoint gets a typed wrapper returning a typed response. Add new endpoints there with explicit types.
- Errors throw `ApiError`; query/mutation `onError` handlers display them via `sonner` toast.

### Forms
- Use `react-hook-form` + `zod`. Define a zod schema, infer the form type, wire `zodResolver`. The `LinkForm` component is the reference.

### Routing
- `react-router-dom` v6. Routes live in `src/App.tsx`. Page components live under `src/pages/` and never import each other.
- Deep-linkable URLs. State that can be in the URL (filters, search, current tab) should be — use `useSearchParams`.

### TypeScript
- `tsc -b` must pass. `tsconfig.app.json` has `strict`, `noUnusedLocals`, `noUnusedParameters` enabled.
- No `any` unless interfacing with an unyielding library type. Prefer `unknown` + narrowing.
- Path alias `@/*` maps to `src/*`.

## MDX

- Real MDX compilation happens **client-side** via `@mdx-js/mdx`'s `evaluate()` in `src/lib/mdx.tsx`. The server returns raw source from `/api/docs/{filename}`.
- Components exposed to MDX are explicitly enumerated in the `mdxComponents` map. Adding a new component for authors means: import it, add it to that map. No magic auto-discovery.
- `remark-gfm` provides tables, strikethrough, task lists; `rehype-highlight` provides syntax highlighting (GitHub theme).
- **Security:** runtime MDX evaluates JSX as code in the viewer's browser, so `POST /api/docs` (and `DELETE`) are **admin-only** (gated via the `RequireAdmin` subrouter). This closes the former unauthenticated-upload hole. Reads stay public.

## Workflows

### Development
- `make dev` runs the Go server (with `air` if installed) and the Vite dev server (`:5173`) concurrently. Vite proxies `/api`, `/query`, and `/auth` to the backend (default `:8080`; override with `VITE_PROXY_TARGET` in `web/frontend/.env.local`).
- Frontend-only: `make frontend-dev`. Backend-only: `go run ./cmd/server` (the committed stub `dist/index.html` will serve a "build the frontend" page until you run `make frontend-build`).

### Build
- `make build` runs `npm run build` then `go build`. The Vite output is embedded via `//go:embed all:dist`.
- A stub `dist/index.html` is committed so `git clone && go build` produces a runnable binary even before the frontend has been built.

### Docker
- Three-stage `Dockerfile`: `node:20-alpine` builds the SPA → `golang:1.21-alpine` builds the binary with the SPA embedded → `alpine:3.18` runtime with only the binary, `docs/`, and the data volume.
- The runtime image must have no `web/` directory. If you see one, the Dockerfile has regressed.

## Cross-cutting principles

1. **Single artifact.** One Go binary serves API, redirects, and SPA. Don't introduce a separate frontend service.
2. **Trust the boundary.** Validate at request edges (`internal/handlers/*`, frontend `lib/api.ts`). Inside the boundary, types are honest.
3. **Reuse over rewrite.** Before adding a component or helper, check `components/ui/`, `lib/utils.ts`, `lib/api.ts`, and the service layer.
4. **Boring tech.** Stick to the existing stack unless there's a concrete reason to add a dependency.
5. **Readable, testable, documented.** Every exported Go function gets a GoDoc-style comment. Every JSON endpoint has at least a smoke test.

## Aspirational (not yet wired)

These are good practices to apply *if and when* the project grows into them — don't shoehorn them into the current codebase.

- **OpenTelemetry** tracing, metrics, and structured logs. Adopt once there's an actual observability backend to ship to (Collector, Jaeger, Prometheus, etc.).
- **Distributed rate-limiting** (Redis-backed). Single-instance deployment doesn't need it.
- **CSRF tokens.** Writes currently rely on `SameSite=Lax` cookies + JSON-only bodies, which is adequate for a same-origin SPA. Add token-based CSRF protection before any cross-origin or multi-tenant public deployment.
- **OAuth / SSO.** The auth layer is email+password today; the service is structured so an OAuth provider can be added as an alternate login path.
- **Retries / circuit breakers / backoff.** Add when external dependencies appear; today there are none beyond SQLite.
