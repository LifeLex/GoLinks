# Architecture

This document describes how GoLinks is wired end-to-end: the layered Go backend, the React/Vite SPA, the way they're glued together into a single binary, and exactly what each HTTP endpoint does.

`README.md` covers what the app is and how to use it. `CLAUDE.md` captures conventions for contributors. This file is the implementation reference — read it when you need to understand or change behaviour.

---

## At a glance

```
┌────────────────────────────────────────────────────────────────────┐
│                         golinks (single binary)                    │
│                                                                    │
│  cmd/server/main.go                                                │
│    └── gorilla/mux router  (Authenticate middleware loads user)    │
│         ├── /query/{path:.*}            → 302 redirect   [public]  │
│         ├── /auth/*  (login/logout/me/status/setup)     [public]  │
│         ├── /api/links  GET [public] · POST [auth]      → JSON     │
│         ├── /api/search (GET)           → JSON          [public]   │
│         ├── /update/    (POST)          → text          [auth]     │
│         ├── /api/docs   GET [public] · POST [admin]     → JSON     │
│         ├── /api/docs/{file} GET [public] · DEL [admin] → JSON     │
│         ├── /api/users  (GET/POST/DEL)  → JSON          [admin]    │
│         └── /*  (catch-all)             → embedded SPA / index.html│
│                                                                    │
│  internal/                                                         │
│    handlers ──▶ service ──▶ repository ──▶ database (SQLite)       │
│       ▲                                          │                 │
│       └── domain models (json + db tagged) ◀────┘                  │
│                                                                    │
│  web/frontend/                                                     │
│    React 18 + TS + Vite + Tailwind + shadcn/ui                     │
│    TanStack Query · react-hook-form+zod · @mdx-js/mdx              │
│    └── dist/  ◀── //go:embed all:dist  (web/frontend/embed.go)     │
│                                                                    │
│  docs/        ── on-disk markdown/MDX (read at runtime)            │
│  data/        ── SQLite database file                              │
└────────────────────────────────────────────────────────────────────┘
```

One process. Two TCP listeners only in development (`:8080` Go, `:5173` Vite proxying back). In production, one listener: `:8080`.

---

## Repository layout

```
cmd/server/main.go                     Entrypoint: config → DB → repos → services → handlers → router
internal/
├── config/config.go                   Env / .env loading; Config struct
├── database/sqlite.go                 sql.DB + schema migrations
├── domain/models.go                   Shortcut, Query, KeywordInfo, PopularQuery, LinkRequest, User, Session (json+db tags)
├── domain/errors.go                   Auth sentinel errors (ErrInvalidCredentials, ErrEmailTaken, …)
├── handlers/handler.go                Redirect + /api/links + /api/search + legacy /update/
├── handlers/document.go               /api/docs CRUD
├── handlers/auth.go                   /auth/* + /api/users; session cookie helpers
├── handlers/middleware.go             Authenticate / RequireAuth / RequireAdmin + UserFromContext
├── handlers/*_test.go                 Table-driven tests with mock services
├── logger/logger.go                   slog wrapper
├── repository/shortcut.go             SQL for linktable (+ tags, search)
├── repository/query.go                SQL for queries (analytics)
├── repository/user.go                 SQL for users
├── repository/session.go              SQL for sessions
└── service/
    ├── link.go                        LinkService: GetLink, UpdateLink, GetRecentQueries, GetAllKeywords, Search
    ├── document.go                    DocumentService: GetDocument, SaveDocument, ListDocuments, DeleteDocument
    └── auth.go                         AuthService: Bootstrap, Login, Authenticate, Logout, CreateUser, ListUsers, DeleteUser
web/frontend/
├── embed.go                           //go:embed all:dist + SPA fallback handler
├── vite.config.ts                     Dev server proxy: /api, /query → :8080
├── src/
│   ├── App.tsx                        Routes
│   ├── main.tsx                       Entry: QueryClientProvider, BrowserRouter, Toaster
│   ├── index.css                      Tailwind + Rams tokens → shadcn HSL vars + prose overrides
│   ├── components/                    Navbar, LinkForm, KeywordTable, RecentQueries, DocUploader, MDXRenderer
│   ├── components/ui/                 shadcn primitives (button, input, card, table, alert, dialog, tabs, sonner, skeleton)
│   ├── pages/                         Home, Setup, DocsList, Doc, NotFound
│   └── lib/
│       ├── api.ts                     Typed fetch wrappers + ApiError
│       ├── mdx.tsx                    @mdx-js/mdx evaluate() + component map
│       └── utils.ts                   cn() helper
docs/                                  Sample markdown/MDX (and uploaded user docs)
.zed/debug.json                        Zed debugger configs (Backend Go, Vite, Chrome)
Dockerfile                             Three-stage: node → go → alpine
Makefile                               dev / build / test / docker / lint
```

---

## Backend layers

The backend follows Clean Architecture. Each layer depends only on the layer below it via interfaces.

### `cmd/server/main.go` — entrypoint

Orchestrates startup in this order (`main.go:24-73`):

1. `config.Load()` — reads `.env` (optional) and env vars (`PORT`, `DATABASE_PATH`, `BASE_URL`, `ENVIRONMENT`, `LOG_LEVEL`).
2. `logger.Initialize(cfg.Logging)` — structured slog logger.
3. `database.NewSQLiteDB(cfg.DatabasePath)` + `database.Migrate(db)` — open connection, run idempotent migrations.
4. Construct repositories (`shortcutRepo`, `queryRepo`).
5. Construct services (`linkService`, `docService`).
6. Construct handlers (`handler`, `docHandler`).
7. Build router; register routes; mount the embedded SPA at the catch-all.
8. Start HTTP server in a goroutine; block on `SIGINT`/`SIGTERM`; graceful shutdown with 30 s timeout.

### `internal/handlers/` — HTTP transport

The thinnest layer: parse requests, call services, encode responses. No business logic. Errors of type `service.InvalidQueryError` are mapped to HTTP 400; everything else is 500.

`writeJSON(w, status, body)` in `internal/handlers/document.go:128` is the canonical JSON encoder used across both files.

### `internal/service/` — business logic

The brain. `LinkService` implements golink resolution semantics including space-splitting and aliasing (more under `/query/` below). `DocumentService` does file I/O and frontmatter parsing.

Services depend on repository **interfaces** (`ShortcutRepository`, `QueryRepository` declared in `service/link.go:15-25`), not concrete types — that's how `mockLinkService` in `handler_test.go` is possible.

### `internal/repository/` — data access

Plain `database/sql` queries. No ORM. Each method takes a `context.Context` and returns domain types defined in `internal/domain/`.

### `internal/database/` — SQLite

Three tables (`internal/database/sqlite.go:28-46`):

- **linktable** — `(id, word, link, user, created_at)`. `user` holds the author's email.
- **queries** — `(query_id, word_id → linktable.id, created_at)`.
- **tags** — `(id, word_id → linktable.id, tag)`, unique on `(word_id, tag)`. Powers tag chips and search.
- **users** — `(id, email, password_hash, role, created_at)`, unique on `lower(email)`.
- **sessions** — `(token_hash PK, user_id → users.id ON DELETE CASCADE, expires_at, created_at)`.

Indexes on `linktable.word`, `queries.word_id`, `queries.created_at`, `tags.tag`, `sessions.user_id`, `sessions.expires_at`. Foreign keys are enabled via the connection string (`?_foreign_keys=on`) — required for the session cascade. Migrations are an ordered `[]string` in `Migrate()`; tests run the same `Migrate` against in-memory DBs so the schema can't drift.

### `internal/domain/` — shared models

POGOs (Plain Old Go Objects) with `json:` and `db:` tags. Used unchanged as both DB row targets and API response bodies — so a schema rename ripples through both layers automatically.

### `web/frontend/embed.go` — SPA bridge

Compile-time `//go:embed all:dist` pulls the Vite build output into the binary. The exported `Handler(reservedPrefixes...)` returns an `http.Handler` that:

- Refuses non-GET/HEAD with 405.
- Refuses any path starting with `api/`, `query/`, or `auth/` (defense in depth — those routes match earlier, but if registration order ever changes, the SPA must not shadow them).
- Serves real files from the embedded FS when they exist (`/assets/*`, `/favicon.ico`).
- Falls back to `index.html` with `Cache-Control: no-cache` for everything else, so React Router can take over on hard refreshes of `/setup`, `/docs/foo`, etc.

There's also a `brokenHandler` that returns 503 with a helpful message if the embedded `dist/` is missing `index.html`. Combined with the committed stub `dist/index.html`, this guarantees `git clone && go build` always produces a runnable binary.

---

## Authentication & authorization

Email + password with server-side sessions. No JWT, no signing secret.

### Storage
- **`users`** — `(id, email, password_hash, role, created_at)`, with a `UNIQUE INDEX on lower(email)` for case-insensitive uniqueness. Passwords are bcrypt-hashed (`golang.org/x/crypto/bcrypt`, cost configurable).
- **`sessions`** — `(token_hash PK, user_id → users.id ON DELETE CASCADE, expires_at, created_at)`. Only the **SHA-256 hash** of the opaque session token is stored, so a database read can't mint a valid cookie. Indexed on `user_id` and `expires_at`.

### Session lifecycle (`service/auth.go`)
1. **Login / setup** verifies the password (with a dummy bcrypt compare on unknown emails to equalize timing), generates a 32-byte `crypto/rand` token, stores `sha256(token)`, and returns the raw token. A fresh token is minted on every login (no session fixation).
2. The raw token rides in an `HttpOnly`, `SameSite=Lax`, `Secure` (production) cookie named `golinks_session`, with `MaxAge = SESSION_TTL_HOURS`.
3. **Authenticate** hashes the cookie value, looks up the session, and (if unexpired) loads the user. Expired sessions are deleted on access. A background goroutine in `main.go` purges expired sessions hourly.

### Bootstrap
On an empty `users` table, `POST /auth/setup` creates the first user as `admin`. Afterwards `Bootstrap` returns `ErrRegistrationClosed` — there is no open self-registration; admins create users via `POST /api/users`.

### Request gating (`handlers/middleware.go`, wired in `main.go`)
- A **global `Authenticate` middleware** (`router.Use`) loads the optional user into request context. It never rejects — an absent/invalid cookie just means anonymous, so public routes keep working. `handlers.UserFromContext(ctx)` retrieves it.
- Two **subrouters** carry guards: `authed` (`RequireAuth` → 401 if anonymous) and `admin` (`RequireAdmin` → 401 anonymous / 403 non-admin). Each `RegisterRoutes` receives the routers it needs and registers reads on the public router, writes on the gated ones. Method-specific routes (GET public, POST authed on the same path) coexist because mux matches by method.
- `getUserID` in `handler.go` now reads the email from context; write handlers run behind `RequireAuth`, so a user is always present there. The author stored on a link (`linktable.user`) is the logged-in user's email.

### CSRF
Writes rely on `SameSite=Lax` cookies plus JSON-only bodies for a same-origin SPA: a cross-site form can't send `application/json` without a preflight, and Lax blocks cross-site POST cookies. No CSRF token today — revisit for cross-origin/multi-tenant deployments.

---

## Frontend architecture

### Entry & routing

`src/main.tsx` mounts `<App />` inside `QueryClientProvider` + `BrowserRouter` and renders `<Toaster>` for sonner.

`src/App.tsx` is the route table:

| Path             | Component       | Notes                                         |
| ---------------- | --------------- | --------------------------------------------- |
| `/`              | `HomePage`      | Form + keyword list + recent queries          |
| `/homepage`      | `HomePage`      | Legacy alias from the template era            |
| `/setup`         | `SetupPage`     | Per-browser instructions in shadcn `Tabs`     |
| `/docs`          | `DocsListPage`  | List + upload + delete                        |
| `/docs/:filename`| `DocPage`       | Fetches raw source, runtime-compiles MDX      |
| `*`              | `NotFoundPage`  | Custom 404 with shortcuts back to home/setup  |

### State management

- **Server state → TanStack Query.** Every fetch goes through a `useQuery` (read) or `useMutation` (write). Mutations call `queryClient.invalidateQueries(['links'])` etc. on success to refresh stale views.
- **URL state → `useSearchParams`.** The `?missing=foo` query param after a failed redirect is read in `HomePage` and shown as a toast, then cleared.
- **Form state → `react-hook-form` + `zod`.** `LinkForm` is the canonical example.
- **Local UI state → `useState`.** Component-scoped, never lifted into a global store.

### API client

`src/lib/api.ts` exports an `api` object with one function per endpoint. Each is typed end-to-end:

```ts
api.listLinks()           : Promise<LinksResponse>
api.createLink({word,link}): Promise<{success: true}>
api.listDocs()            : Promise<{documents: DocumentInfo[]}>
api.getDoc(filename)      : Promise<DocumentSource>
api.uploadDoc(file: File) : Promise<{success: true; filename; url}>
api.deleteDoc(filename)   : Promise<{success: true}>
```

Non-2xx responses throw `ApiError` carrying the body text + status code.

### Styling

`src/index.css`:

- Imports `@fontsource/inter` (300/400/500/600), `@fontsource/jetbrains-mono` (400/500), and `highlight.js/styles/github.css`.
- `@tailwind base/components/utilities`.
- Defines shadcn HSL CSS variables under `:root`: `--background`, `--foreground`, `--primary` (Braun orange), `--accent` (functional blue), `--destructive`, `--muted`, etc. These map onto Dieter Rams-inspired tokens — see comments in the file for the original hex values.
- Prose overrides under `@layer base` make MDX-rendered docs match the rest of the UI (orange-on-hover links, mono code blocks, etc.).

`tailwind.config.ts` extends Tailwind with the shadcn token mapping (`bg-primary` → `hsl(var(--primary))`).

### MDX pipeline

```
Doc page mounts
  └── api.getDoc(filename) → { source, type, metadata }
       └── <MDXRenderer source={source} />
            └── compileMDX(source)        // src/lib/mdx.tsx
                 └── @mdx-js/mdx evaluate()
                      ├── remark-gfm           (tables, task lists, strikethrough)
                      ├── rehype-highlight     (syntax highlighting)
                      └── useMDXComponents → mdxComponents map
                           ├── Alert, AlertTitle, AlertDescription
                           ├── Card, CardHeader, CardTitle, CardDescription, CardContent
                           ├── Button
                           ├── Tabs, TabsList, TabsTrigger, TabsContent
                           └── table/thead/tbody/tr/th/td → shadcn Table primitives
```

The whole pipeline runs in the browser. The Go server never sees compiled HTML — it just serves the raw `.md` / `.mdx` source.

---

## Build & distribution

### Development

`make dev` runs the Go server (via `air` if installed, else `go run`) and the Vite dev server (`:5173`) concurrently. Vite's dev server has a proxy (`vite.config.ts`) that forwards `/api/*`, `/query/*`, and `/auth/*` to the backend (default `http://localhost:8080`; override with `VITE_PROXY_TARGET` in `web/frontend/.env.local`), so the React app can call relative URLs and hit the Go backend without CORS gymnastics.

For breakpoint debugging, see `.zed/debug.json` — three configs: Backend (Go via Delve), Frontend dev server (Vite via Node), Frontend (Chrome).

### Production

`make build` runs:

1. `npm ci && npm run build` inside `web/frontend/` → produces `web/frontend/dist/`.
2. `go build -o build/golinks ./cmd/server` — the `//go:embed all:dist` directive in `web/frontend/embed.go` pulls the dist into the binary.

Result: a single ~14 MB binary that needs only the `docs/` directory and the SQLite db file at runtime.

### Docker

`Dockerfile` is three-stage:

1. `node:20-alpine` — installs `web/frontend/package*.json`, runs `npm ci`, then `npm run build`. Output: `/app/web/frontend/dist/`.
2. `golang:1.21-alpine` — copies the dist over the top of any committed stub, runs `CGO_ENABLED=1 go build` (CGO needed for the SQLite driver).
3. `alpine:3.18` — final runtime. Copies the binary and the `docs/` directory. **No `web/`** in the runtime image. Drops to a non-root `golinks` user. Exposes 8080. SQLite db lives in `/app/data/`.

The `HEALTHCHECK` hits `http://localhost:8080/` every 30 s.

---

## Endpoint reference

Routes are registered by each handler's `RegisterRoutes(public, gated)` method (links, docs, auth), with the global `Authenticate` middleware, the `authed`/`admin` subrouters, and the SPA catch-all wired in `cmd/server/main.go`. Match order matters — gorilla/mux matches in registration order, but method-specific routes (GET-public, POST-authed on the same path) coexist regardless of order. Each endpoint below notes its access level: **[public]**, **[auth]**, or **[admin]**.

### `GET /query/{path:.*}` — golink redirect

The contract that justifies a server-side rendered handler. Browser search-engine integrations issue plain HTTP requests; they don't run JavaScript, so this MUST be a 302 from the Go server.

**Flow** (`handler.go:RedirectHandler`):

1. Strip trailing slash from the captured `path`.
2. Call `linkService.GetLink(ctx, path, "")`.
3. On success → `302` to the resolved URL.
4. On `InvalidQueryError` → `302` to `${BASE_URL}/?missing=<path>` (the SPA picks up the `missing` param and shows a toast).
5. On any other error → `500`.

**Resolution semantics** (`service/link.go:GetLink`):

- Look up `word` in `linktable`. If found:
  - Log a hit in `queries` (best-effort; logging failure does not fail the request).
  - If the stored `link` is itself a keyword (not `http(s)://...`), recurse → enables aliases.
  - If the stored `link` contains `{*}`, substitute the `searchTerm` (URL-encoded).
  - Return the URL.
- If not found *and* `word` contains spaces, peel the last token off and treat it as a search term (`go google cats` → look up `google cats`, fail, then `google` with searchTerm `cats`).
- If still not found → return `InvalidQueryError`.

### Auth endpoints (`handlers/auth.go`)

- **`GET /auth/status`** [public] — `{needs_setup, authenticated, user?}`. Drives the SPA bootstrap (`AuthProvider`).
- **`GET /auth/me`** [public] — `{authenticated, user?}`.
- **`POST /auth/setup`** [public] — first-run only. Body `{email, password}`; creates the first user as `admin`, sets the session cookie. `403` if a user already exists.
- **`POST /auth/login`** [public] — body `{email, password}`; sets the session cookie on success, `401` on bad credentials (unknown email and wrong password are indistinguishable).
- **`POST /auth/logout`** [public] — deletes the session and clears the cookie (idempotent).
- **`GET /api/users`** [admin] — `{users: [...]}`.
- **`POST /api/users`** [admin] — body `{email, password, role}`; `409` on duplicate email, `400` on weak password.
- **`DELETE /api/users/{id}`** [admin] — `409` (`ErrLastAdmin`) if it would remove the only admin; `404` if absent.

### `GET /api/links` — list keywords + recent queries [public]

`handler.go:ListLinks`. Returns everything the homepage needs in one call.

```json
{
  "keywords":      [ { "word": "...", "link": "...", "created_at": "..." }, ... ],
  "recent_queries":[ { "count": 5, "word": "...", "link": "..." }, ... ],
  "base_url":      "http://localhost:8080"
}
```

- `keywords` ← `linkService.GetAllKeywords()` → all rows from `linktable` filtered to URL-shaped links (skipping aliases).
- `recent_queries` ← `linkService.GetRecentQueries()` → top 20 by count over the last 3 days, joined back to `linktable` for the URL.
- `base_url` ← `cfg.BaseURL`, used by the SPA to display the search-engine URL on the home and setup pages.

### `POST /api/links` — create or update a link

`handler.go:CreateLink`. JSON body: `{"word":"...","link":"..."}`.

1. Decode JSON; trim whitespace on both fields.
2. Call `linkService.UpdateLink`, which validates:
   - Non-empty `word` and `link`.
   - `word` does not end in `/`.
   - `link` starts with `http://` or `https://`.
   - `link != word` (no self-loops).
3. Insert a new row into `linktable`.
4. Return `{"success": true}` (200) or `400` with the validation message.

### `POST /update/` — legacy form-encoded create

`handler.go:UpdateLinkLegacy`. Same semantics as `POST /api/links`, but accepts `application/x-www-form-urlencoded` (`word=...&link=...`) and returns plain text `Link added successfully!`. Kept so anyone who configured their browser against the pre-migration `/update/` endpoint still works.

### `GET /api/docs` — list documents

`document.go:ListDocuments`. Reads `docs/`, returns one entry per `.md` / `.mdx` file. For each file, peeks at the first lines to extract `title` and `description` from YAML frontmatter (`service/document.go:peekFrontmatter`); falls back to the filename without extension. Type is inferred from the extension.

```json
{ "documents": [ { "title": "...", "description": "...", "type": "markdown|mdx", "path": "sample.md" } ] }
```

### `GET /api/docs/{filename}` — fetch raw source

`document.go:GetDocument`. Returns the full file contents (frontmatter included) plus parsed metadata.

```json
{
  "source":   "---\ntitle: ...\n---\n# ...",
  "type":     "markdown|mdx",
  "metadata": { "title": "...", "description": "...", "type": "...", "path": "...", "metadata": { ...full frontmatter map... } }
}
```

If the URL omits the extension (`/api/docs/sample`), the handler tries `.md` first, then `.mdx`. 404 if neither exists. The client compiles MDX in the browser (`web/frontend/src/lib/mdx.tsx`), so the server never renders to HTML.

### `POST /api/docs` — upload a document [admin]

`document.go:UploadDocument`. `multipart/form-data` with field `file`.

1. Parse multipart form (10 MB limit).
2. Reject files whose name doesn't end in `.md` or `.mdx`.
3. Sanitize the filename via `filepath.Base` (no path traversal) and write into `docs/`.
4. Return `{success, filename, message, url}` (the `url` is `/docs/{filename}` — a SPA route, not an API route).

**Admin-gated.** Runtime MDX compiles JSX in the viewer's browser, so upload is restricted to admins via the `RequireAdmin` subrouter — this closes the former unauthenticated stored-XSS/RCE vector.

### `DELETE /api/docs/{filename}` — delete a document [admin]

`document.go:DeleteDocument`. Sanitize via `filepath.Base`, `os.Remove`. Returns `{success, message}` or 500.

### `GET /` and `GET /<anything-else>` — embedded SPA

The catch-all (`cmd/server/main.go`) hands the request to `frontend.Handler("api", "query", "auth")`. Behaviour described in the **`web/frontend/embed.go`** section above.

---

## Configuration

Loaded by `internal/config/config.go`. All env vars optional; defaults shown.

| Variable        | Default                  | Used by                                                  |
| --------------- | ------------------------ | -------------------------------------------------------- |
| `PORT`          | `8080`                   | `cmd/server/main.go` server bind                         |
| `DATABASE_PATH` | `golinks.db`             | `database.NewSQLiteDB`                                   |
| `BASE_URL`      | `http://localhost:8080`  | Returned in `/api/links` and used in 302 fallback target |
| `ENVIRONMENT`   | `development`            | Logged on startup; sets the `COOKIE_SECURE` default      |
| `LOG_LEVEL`     | `info`                   | `logger.Config.Level`                                    |
| `SESSION_TTL_HOURS` | `720`                | Session lifetime + cookie `MaxAge` (`config.AuthConfig`) |
| `COOKIE_SECURE` | prod `true` / dev `false`| Session cookie `Secure` flag                             |
| `BCRYPT_COST`   | `12`                     | bcrypt work factor for password hashing                  |
| `MIN_PASSWORD_LEN` | `8`                   | Minimum password length                                  |

`.env` at the repo root is auto-loaded if present (godotenv). See `env.example`.

---

## Security & known TODOs

- **Doc uploads are admin-gated** (resolved). `POST`/`DELETE /api/docs` require the admin role, closing the former unauthenticated stored-XSS/RCE vector from runtime MDX.
- **Authentication is implemented** (resolved). `getUserID` reads the user from request context; `linktable.user` holds the author's email.
- **No CSRF token** on writes. Mitigated by `SameSite=Lax` cookies + JSON-only bodies for the same-origin SPA; adequate for single-instance use. Add token-based CSRF before any cross-origin or multi-tenant public deployment.
- **Email + password only.** No OAuth/SSO yet; the `AuthService` is structured so a provider can be added as an alternate login path.
- **No account-level rate limiting** on `/auth/login`. Add brute-force protection (per-IP/account throttling) before public exposure.

## Future / aspirational

The `pkg/`, `api/`, `configs/`, and `test/` directories from typical Go layouts are **not** present — this repo doesn't need them yet. Add them only when there's a concrete reason (shared utilities consumed by another binary, gRPC API definitions, etc.).

OpenTelemetry, distributed rate-limiting, retry/backoff for external calls, circuit breakers — none are wired today and none should be added speculatively. The current scope is a single-instance tool with one external dependency (SQLite).
