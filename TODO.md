# TODO

Roadmap for GoLinks. Items are grouped by theme and ordered roughly by dependency — top sections unblock lower ones. Each item has a short scope line so Claude (or future-you) can pick one up without re-deriving context. See `ARCHITECTURE.md` for the underlying design and `CLAUDE.md` for conventions.

## Suggested order

1. **MCP server (Phase 1)** — independent, ships immediate value, can run before broader auth via a shared bearer token.
2. **Authentication** — unblocks the MDX upload security gap and is required for everything below.
3. **Authorization & roles** — depends on auth.
4. **Admin features** (edit/delete UI, request/approval flow) — depend on roles.
5. **Tooling** (migrations library) and **independent features** (dark mode, doc analytics) — do whenever.

---

## MCP server — Phase 1 (GoLinks-native)

The aim: let agents resolve, search, read, and create golinks via MCP. **Phase 1 only exposes GoLinks' own data (`linktable` + `docs/`).** Federated search across third-party sources is intentionally a Phase 2 decision, gated on real usage gaps observed after Phase 1 ships — see *Open questions* below.

- [ ] **Add `internal/adapters/mcp/` server package.** Use `github.com/mark3labs/mcp-go` for protocol plumbing (HTTP-streamable transport, since this is team-shared). The MCP server is an inbound adapter alongside `httpapi/`; reuse `core/links.Service` and `core/docs.Service` — no business logic in the adapter.
- [ ] **`MCP_TOKEN` bearer middleware.** Read from env, validate `Authorization: Bearer <token>` on every MCP request. Reject with 401 on miss. Document in `env.example`.
- [ ] **Mount at `/mcp`** in `cmd/server/main.go` after `/api/*` and before the SPA catch-all. Add `mcp` to the `frontend.Handler` reserved-prefix list.
- [ ] **FTS5 index over `linktable`.** New migration: virtual table mirroring `word + link`, with INSERT/UPDATE/DELETE triggers to keep it in sync. Required for `search_golinks`.
- [ ] **FTS5 index over `docs/`.** Built on startup, updated on upload and delete in `DocumentService`. Strip frontmatter before indexing. Required for `search_docs`.
- [ ] **Tool: `resolve_golink(word: string)`** — wraps `core/links.Service.GetLink`. Returns `{url}` on hit, error on miss.
- [ ] **Tool: `search_golinks(query: string, limit?: int = 10)`** — full-text search. SQLite uses FTS5 (virtual table + triggers); Postgres uses `tsvector` + GIN. Both implementations live behind a new port on the persistence layer. Returns `[{word, link, score}]`.
- [ ] **Tool: `list_golinks(limit?: int = 100, offset?: int = 0)`** — wraps `core/links.Service.GetAllKeywords` with pagination. Returns `[{word, link, created_at}]`.
- [ ] **Tool: `search_docs(query: string, limit?: int = 10)`** — full-text search over the docs corpus. SQLite FTS5 / Postgres tsvector index built and maintained from the docs/ folder. Returns `[{filename, title, snippet, score}]`.
- [ ] **Tool: `fetch_doc(filename: string)`** — wraps `core/docs.Service.GetDocument`. Returns `{source, type, metadata}`.
- [ ] **Tool: `create_golink(word: string, url: string)`** — wraps `core/links.Service.UpdateLink`. Returns `{success}` or validation error.
- [ ] **Smoke tests in `internal/adapters/mcp/server_test.go`** — table-driven, with the same mock pattern as `httpapi/links_handler_test.go`. Cover: token rejection, each tool happy path, search with no results, validation errors.
- [ ] **Update `ARCHITECTURE.md`** with the `/mcp` endpoint, tool catalog, and the Phase 1/Phase 2 scope decision. Update `CLAUDE.md`'s endpoint list and add a short "MCP conventions" section.
- [ ] **Update `README.md`** with a "Connecting an agent" section: example MCP client config (Claude Desktop / Claude Code) using `MCP_TOKEN`.

## Authentication & authorization

- [ ] **Authentication.** Pick an approach and implement it: (a) proprietary email+password with bcrypt, (b) OAuth via GitHub/Google, (c) shared session token. Today `userIDFromRequest` in `internal/adapters/httpapi/helpers.go` returns `"DefaultUser"` unconditionally — replace it with a real identity lookup. Blocks the runtime MDX upload risk flagged in `ARCHITECTURE.md` and `CLAUDE.md`.
- [ ] **Authorization with roles.** Two roles to start: `admin` (full CRUD on golinks and docs) and `user` (read, search, propose). Gate `POST /api/links`, `POST /api/docs`, `DELETE /api/docs/*`, and `create_golink` MCP tool on `admin`. _Depends on authentication._

## Tooling

- [x] ~~**Switch to `goose` (or `golang-migrate`).**~~ Done 2026-04-28 — Goose adopted. Migrations live in a single folder at `internal/adapters/persistence/migrations/` as Go functions that delegate schema work to GORM, so dialect translation is handled by the ORM rather than maintained as parallel SQL files.

## Features

- [ ] **Dark mode.** shadcn theming is already token-driven — add a `.dark` block in `web/frontend/src/index.css` overriding the HSL variables. Toggle via a `<button>` in the navbar; persist in `localStorage`; respect `prefers-color-scheme` on first visit. (Replaces the older "theme customization" item — full per-user themes is overkill at this stage.)
- [ ] **Document access analytics.** The `queries` table tracks golink hits. Mirror the pattern for docs: add a `doc_views` table (new Goose migration in both dialects), record a row in `httpapi.DocsHandler.Get`, expose `GetPopularDocs` via `core/docs.Service` + `/api/docs/popular` endpoint. Surface "Recently viewed" on `/docs` in the SPA.
- [ ] **Edit & delete UI for golinks.** Today the homepage only adds. Add row-level edit and delete actions on the keyword table in `web/frontend/src/components/KeywordTable.tsx`. Backend: new `PATCH /api/links/{word}` and `DELETE /api/links/{word}` handlers, plus matching methods on `core/links.Service` and `core/links.Repository`. _Depends on authorization (admin-only)._
- [ ] **Keyword-request workflow.** Non-admins propose new golinks; admins approve or reject. Either add a `status` column to `linktable` (`proposed` / `approved` / `rejected`) or a new `link_requests` table that promotes to `linktable` on approval. New API endpoints; new admin view in the SPA. _Depends on authorization._

## Open questions (not yet actionable)

- **MCP Phase 2 — federated search.** Whether to add adapters for third-party sources (Confluence, Notion, Drive, Slack) is intentionally deferred. Decide after Phase 1 has been used for a few weeks: track which agent queries Phase 1 *can't* answer, and pick the highest-demand source first. Designing a generic `Source` interface before having two real adapters is premature.
- **Deployment target.** Now that CGO is gone (pure-Go SQLite + Postgres) the binary fits anywhere: Cloud Run, App Engine, Fly.io, plain VMs, App Service. SQLite mode needs a writable persistent volume; Postgres mode pairs with Cloud SQL / RDS / Neon / Supabase. Pick a target before adding deployment automation.

## Decisions (removed from prior list)

Recording these so they don't reappear without a fresh discussion:

- ~~**Postgres support — removed.**~~ **Reversed 2026-04-28.** Postgres is now supported alongside SQLite via GORM dialect dispatch; SQLite remains the default. Selection via `DATABASE_DRIVER`. See `ARCHITECTURE.md` for the connection layer.
- ~~**ORM — rejected.**~~ **Reversed 2026-04-28: GORM adopted.** Rationale: SQLite/Postgres dual support without per-dialect query code; explicit FK constraints; less hand-rolled boilerplate. Trade-off accepted: less direct control over generated SQL.
- ~~**Render Markdown and MDX.**~~ Done — markdown via `remark-gfm`, real MDX via runtime `@mdx-js/mdx evaluate()` in `web/frontend/src/lib/mdx.tsx`.
- ~~**Generic "user administration".**~~ Subsumed by **Authentication** + **Authorization** above.
- ~~**Deploy on Google App Engine specifically.**~~ See *Open questions*.
