# Roadmap

The strategic view of where GoLinks is going and why. For the tactical, pick-up-and-go task list, see [`TODO.md`](./TODO.md). For the current implementation, see [`ARCHITECTURE.md`](./ARCHITECTURE.md).

## Positioning

GoLinks is an open-source, self-hosted golinks tool. The space already has incumbents (Trotto, YOURLS, Google's reference implementation), so GoLinks needs a reason to exist beyond "another redirector." We have three candidate wedges:

- **AI-native** — LLM-assisted resolution, natural-language link creation, and agent interfaces (MCP). The first golinks tool built for the agent era.
- **Unified knowledge base** — the MDX docs feature becomes the headline: semantic search, docs-as-fallback when a keyword misses, golinks as bookmarks into docs.
- **Best single-binary DX** — zero-config self-host, clean Go idioms, opinionated defaults. Craft as the differentiator (the Caddy/SQLite school).

**We have not committed to one wedge.** We are a pre-release, long-term project not yet in daily use, so we have no gut feel and no user signal to pick from. The roadmap is designed to *earn* that decision rather than guess it.

## Strategy: foundations, then let data pick the wedge

1. Build the **foundations** every good OSS golinks tool needs, regardless of wedge, and ship a polished **v1.0**.
2. Ship **one small tracer-bullet per wedge** ("breadcrumbs") and watch what released users actually adopt.
3. **Double down** on whichever wedge gets traction. Kill or freeze the others honestly.

This keeps the project focused while deferring the wedge decision to the point where we have evidence.

## Stack policy

Default remains **single Go binary + SQLite** — the property that makes self-hosting trivial. We will add external services (Postgres, Redis, a vector store) **only when a feature genuinely earns it**, and always behind an opt-in so the zero-dependency default survives. AI features bring-your-own-API-key by default.

---

## Phase 1 — Foundations (blocks v1.0)

Everything needed to be a credible OSS golinks tool. Wedge-neutral.

- **Search + tags** — wire up the dormant `tags` table; FTS5 full-text index over links; `/api/search`; search UI. *(Highest leverage: also the substrate for the MCP and docs-RAG wedges.)*
- **Authentication** — replace the hardcoded `getUserID → "DefaultUser"`. Closes the unauthenticated-MDX-upload risk.
- **Authorization** — `admin` / `user` roles gating writes.
- **Edit & delete UI** for golinks (admin-only).
- **Browser extension / setup polish** — make the first-run experience good (the thing that decides whether a self-hoster sticks).
- **Link analytics** — surface the data already in the `queries` table; mirror it for docs.
- **Project hygiene** — versioned migrations (`goose`), dark mode, hardening, CI polish, contributor docs.

## Phase 2 — Wedge breadcrumbs (after v1.0)

Ship together; instrument; compare adoption.

- **AI breadcrumb** — MCP server exposing resolve/search/read/create over GoLinks' own data (already speced in `TODO.md`).
- **Docs breadcrumb** — semantic fallback: when a keyword misses, search `docs/` and offer the top hit instead of a bare 404.
- **DX breadcrumb** — one-command install + a genuinely zero-config first run (sample data, auto-generated config, health page).

## Phase 3 — Wedge expansion (post-data, open-ended)

Driven by Phase 2 signal. Illustrative directions per wedge:

- **AI** — natural-language link creation, fuzzy/LLM resolution, federated MCP search (Confluence/Notion/Drive/Slack), link-rot detection with suggested fixes.
- **Docs** — embeddings + vector search (sqlite-vec or a real vector DB), RAG answers over docs, doc versioning.
- **DX** — multi-arch releases, Helm chart, Terraform module, polished admin TUI.

## Cross-cutting (slot in opportunistically)

- Multi-placeholder / named-parameter link templates (beyond a single `{*}`).
- **Postgres backend (planned).** We intend to move to Postgres to unlock multi-instance and better search. The repository interfaces (`ShortcutRepository`, `QueryRepository`) already isolate storage, so the blast radius is the repository + database layers only. The cost is losing the zero-dependency single-binary default, so this is gated on multi-instance becoming a real need — but it's a *when*, not an *if*. Two follow-ons ride with it:
  - **Native Postgres full-text search** replaces the interim `LIKE`-based search (`tsvector`/`tsquery` + GIN, optionally `pg_trgm` for typo tolerance). No build tags, unlike SQLite FTS5. The interim `LIKE` search is deliberately written to be portable so this is a clean swap.
  - **Data-access tooling decision** (see decision log) — reconsider the no-ORM stance when we do the migration, *not before*.
- OpenTelemetry — adopt once there's a backend to ship to.

---

## Decision log

- **Wedge: undecided by design.** Revisit after Phase 2 ships and produces adoption data.
- **Search: interim `LIKE`/`ILIKE`, native FTS later.** SQLite FTS5 is intentionally skipped — it needs a `sqlite_fts5` build tag threaded through every build/test/run invocation, and the work would be thrown away on the Postgres move. `LIKE` substring search is portable across SQLite and Postgres and is instant at our data scale. Upgrade to the destination DB's native FTS only if search quality demands it.
- **Postgres: planned (gated on multi-instance).** No longer "deferred indefinitely" — the intent is to migrate. Single-binary + SQLite remains the default until multi-instance is a real need. The repository interfaces already isolate storage, so it's a repo/database-layer change, not an architecture change.
- **Data-access tooling: revisit at Postgres-migration time.** Today's convention is plain `database/sql` (no ORM). The Postgres move reopens this — the placeholder-dialect rewrite (`?` → `$1`) is the trigger. Leading candidate is **`sqlc`** (SQL-first, generates type-safe Go, handles the dialect) over a heavyweight ORM (GORM/ent/Bun), because it preserves the "boring tech / SQL we can read" ethos. **Not decided** — do not adopt anything now; choose when the migration is actually scoped.
