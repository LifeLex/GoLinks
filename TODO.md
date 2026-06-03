# TODO

Tactical task list for GoLinks, ordered to match the phases in [`ROADMAP.md`](./ROADMAP.md). Each item has a short scope line so Claude (or future-you) can pick one up without re-deriving context. See `ARCHITECTURE.md` for the underlying design and `CLAUDE.md` for conventions.

Phase 1 ships v1.0. Phase 2 ships one tracer-bullet per wedge to gather adoption signal. Phase 3 is gated on that signal — don't start it yet.

---

## Phase 1 — Foundations (blocks v1.0)

### Search + tags  ← **start here**

The highest-leverage foundation: it makes the tool browsable past ~50 links, activates the dormant `tags` table, and the FTS5 index it builds is reused by the MCP and docs-RAG breadcrumbs in Phase 2.

- [ ] **FTS5 index over `linktable`.** New migration: a virtual table mirroring `word + link` (+ `tag` once tags land), with INSERT/UPDATE/DELETE triggers to keep it in sync.
- [ ] **Wire up the `tags` table.** It exists in the schema (`internal/database/sqlite.go`) but nothing reads or writes it. Add repository methods (`AddTag`, `GetTags`, `GetByTag`) and surface tags on the link create/edit path.
- [ ] **`GET /api/search?q=&limit=`** — FTS5 query across word + link + tags. Returns `[{word, link, tags, score}]`. New handler + service method; validate/trim `q` at the boundary.
- [ ] **Search UI.** A search box on the homepage that filters the keyword table via the new endpoint. Use `useSearchParams` so `?q=` is deep-linkable (per `CLAUDE.md` routing convention).
- [ ] **Tests.** Table-driven: FTS sync on insert/update/delete, search ranking, empty-query and no-results paths.

### Authentication & authorization

- [x] **Authentication.** Email+password with bcrypt; server-side sessions (SHA-256-hashed opaque token in an HttpOnly `SameSite=Lax` cookie). First user on an empty DB becomes admin via `/auth/setup`; registration is closed thereafter. `getUserID` now reads the user from request context (set by the `Authenticate` middleware). Closes the runtime MDX upload risk.
- [x] **Authorization with roles.** `admin` and `user` roles. A global `Authenticate` middleware loads the optional user; `RequireAuth`/`RequireAdmin` subrouters gate writes. Reads stay public; `POST /api/links` + legacy `/update/` require auth; `POST/DELETE /api/docs` require admin. Minimal admin user management at `/api/users` + `/admin/users`.

### Admin & management UI

- [x] **Edit & delete UI for golinks.** Row-level edit (target + tags) and delete actions in `KeywordTable.tsx` (gated on auth, shown to any logged-in user — consistent with create). Backend: `PATCH /api/links/{word}` (appends a new revision — latest per word wins) and `DELETE /api/links/{word}` (transactional delete of the word's rows + dependent tags/queries; idempotent), both on the `authed` subrouter. _Rename is not yet supported (edit changes target/tags only); it'd need delete+recreate._
- [ ] **Keyword-request workflow.** Non-admins propose new golinks; admins approve or reject. Add a `status` column to `linktable` (`proposed`/`approved`/`rejected`) or a `link_requests` table that promotes on approval. New API endpoints; new admin view in the SPA. _Depends on authorization._

### First-run experience

- [ ] **Setup / onboarding polish.** Make the first run good: sample links seeded on an empty DB, the search-engine setup instructions front-and-center, a health/status page. This is what decides whether a self-hoster sticks.
- [ ] **(Stretch) Browser extension.** A thin extension that registers the `go` keyword automatically instead of manual per-browser search-engine config.

### Analytics

- [ ] **Surface golink analytics.** The `queries` table already records hits — expose trends (top links, recent activity) beyond the current "recent queries" list.
- [ ] **Document access analytics.** Mirror the pattern for docs: add a `doc_views` table (or extend `queries`), record a row in `DocumentHandler.GetDocument`, expose `GetPopularDocs` via service + `/api/docs/popular`. Surface "Recently viewed" on `/docs`.

### Project hygiene

- [ ] **Versioned migrations.** Switch from the inline string slice in `internal/database/sqlite.go:Migrate` to `goose` (or `golang-migrate`). Matters once auth tables + analytics land.
- [x] **Dark mode.** `.dark` token block in `index.css`; `ThemeToggle` in the navbar persists to `localStorage`; an inline script in `index.html` sets the class pre-paint (respects `prefers-color-scheme` on first visit). _Note: the navbar uses `bg-foreground`, so it inverts with the theme — tweak if an always-dark bar is preferred. The `highlight.js` code theme is still the light GitHub theme in dark mode (follow-up)._
- [x] **Contributor docs + login rate-limiting.** `CONTRIBUTING.md`, `.github` issue/PR templates, and per-IP login throttling (`internal/ratelimit`, 429 on `/auth/login` + `/auth/setup`). _Remaining hardening: write rate-limiting beyond login._

---

## Phase 2 — Wedge breadcrumbs (after v1.0 ships)

One small tracer-bullet per wedge, shipped together, instrumented to compare adoption. **Do not over-build any of these** — they exist to gather signal.

### AI breadcrumb — MCP server (GoLinks-native)

Let agents resolve, search, read, and create golinks via MCP over **GoLinks' own data only** (`linktable` + `docs/`). Federated third-party search is a Phase 3 decision (see *Open questions*).

- [ ] **Add `internal/mcp/` server package.** Use `github.com/mark3labs/mcp-go` for protocol plumbing (HTTP-streamable transport, since this is team-shared). Reuse `LinkService` and `DocumentService` — no business logic in the MCP layer.
- [ ] **`MCP_TOKEN` bearer middleware.** Read from env, validate `Authorization: Bearer <token>` on every MCP request. 401 on miss. Document in `env.example`.
- [ ] **Mount at `/mcp`** in `cmd/server/main.go` after `/api/*` and before the SPA catch-all. Add `mcp` to the `frontend.Handler` reserved-prefix list.
- [ ] **FTS5 index over `docs/`.** Built on startup, updated on upload and delete in `DocumentService`. Strip frontmatter before indexing. (The `linktable` FTS5 index already exists from Phase 1.)
- [ ] **Tool: `resolve_golink(word)`** — wraps `LinkService.GetLink`. Returns `{url}` on hit, error on miss.
- [ ] **Tool: `search_golinks(query, limit=10)`** — reuses the Phase 1 FTS5 search.
- [ ] **Tool: `list_golinks(limit=100, offset=0)`** — wraps `LinkService.GetAllKeywords` with pagination.
- [ ] **Tool: `search_docs(query, limit=10)`** — FTS5 over the docs index.
- [ ] **Tool: `fetch_doc(filename)`** — wraps `DocumentService.GetDocument`.
- [ ] **Tool: `create_golink(word, url)`** — wraps `LinkService.UpdateLink`. _Admin-gated once auth lands._
- [ ] **Smoke tests in `internal/mcp/server_test.go`** — same mock pattern as `handler_test.go`. Cover token rejection, each tool happy path, no-results, validation errors.
- [ ] **Docs.** `ARCHITECTURE.md` (the `/mcp` endpoint + tool catalog), `CLAUDE.md` (endpoint list + "MCP conventions"), `README.md` ("Connecting an agent" with example client config).

### Docs breadcrumb — semantic fallback

- [ ] **Docs-as-fallback on a missed keyword.** When `/query/{word}` misses, run a docs search and, if there's a strong hit, redirect to it (or to a disambiguation page) instead of the bare `?missing=` toast. Reuses the Phase 1/MCP FTS5 docs index — keep it keyword-based for the breadcrumb; embeddings are Phase 3.

### DX breadcrumb — zero-config install

- [ ] **One-command install + zero-config first run.** A single `curl | sh` (or `docker run`) that produces a working instance with sample data, an auto-generated config, and a clear health page. Measures whether "easiest to self-host" is the wedge people care about.

---

## Phase 3 — Wedge expansion (gated on Phase 2 signal — do not start yet)

Pick up only after Phase 2 adoption data points to a winner. Illustrative:

- **AI:** natural-language link creation, fuzzy/LLM resolution, federated MCP search, link-rot detection with suggested fixes.
- **Docs:** embeddings + vector search (sqlite-vec or a real vector DB), RAG answers over docs, doc versioning.
- **DX:** multi-arch releases, Helm chart, Terraform module, admin TUI.

---

## Open questions (not yet actionable)

- **MCP Phase 3 — federated search.** Adapters for Confluence/Notion/Drive/Slack are deferred. Decide after the MCP breadcrumb has run for a few weeks: track which agent queries it *can't* answer, pick the highest-demand source first. Designing a generic `Source` interface before two real adapters exist is premature.
- **Deployment target.** Single-binary + SQLite fits Cloud Run (mounted volume), Fly.io, or Docker-on-VM. App Engine Standard is a poor fit (CGO via `mattn/go-sqlite3`); Cloud Run / App Engine Flexible work but the persistent-disk story needs thought. Pick a target before adding deployment automation.

## Decisions (recorded so they don't reappear without fresh discussion)

- **Wedge: undecided by design.** Revisit after Phase 2. See `ROADMAP.md`.
- **Search: interim `LIKE`/`ILIKE`, native FTS later.** SQLite FTS5 skipped — it needs a `sqlite_fts5` build tag in every build/test/run path and would be discarded on the Postgres move. `LIKE` is portable (SQLite `LIKE` ↔ Postgres `ILIKE`) and fast enough at this scale. Write the interim search to keep the swap clean.
- **Postgres: planned, gated on multi-instance.** *(Changed from "deferred, not rejected" — we now intend to migrate.)* Single-binary + SQLite stays the default until multi-instance is real. Repository interfaces already isolate storage → repo/database-layer change, not architecture. Rides with it: native Postgres FTS (replaces interim `LIKE`) and the data-access tooling decision below.
- **Data-access tooling: revisit at Postgres-migration time.** Plain `database/sql` (no ORM) is today's convention and stays for now. The migration's `?`→`$1` placeholder rewrite is the trigger to reconsider. Leading candidate **`sqlc`** (SQL-first, type-safe codegen, dialect-aware) over a full ORM. **Do not adopt now** — decide when the migration is scoped.
- ~~**Render Markdown and MDX.**~~ Done — markdown via `remark-gfm`, real MDX via runtime `@mdx-js/mdx evaluate()` in `web/frontend/src/lib/mdx.tsx`.
- ~~**Generic "user administration".**~~ Subsumed by Authentication + Authorization.
