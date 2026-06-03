# Contributing to GoLinks

Thanks for your interest in improving GoLinks. This guide covers the dev setup, the checks your change must pass, and where to look for conventions.

## Project orientation

- **`README.md`** — what GoLinks is and how to run it.
- **`ARCHITECTURE.md`** — how it's wired end-to-end (backend layers, SPA, endpoints).
- **`AUTH.md`** — the authentication/session-cookie flow.
- **`CLAUDE.md`** — coding conventions (Go + frontend). Please read before sending a PR.
- **`ROADMAP.md`** / **`TODO.md`** — where the project is headed and what's up for grabs.

## Prerequisites

- Go 1.21+
- Node 20+
- A CGO toolchain (the SQLite driver needs it)

## Development

```bash
cp env.example .env          # optional; tweak PORT etc. if 8080 is taken
make dev                     # Go backend + Vite dev server (:5173) together
```

The Vite dev server proxies `/api`, `/query`, and `/auth` to the Go backend (default `:8080`). If `:8080` is in use, set `PORT` in `.env` and `VITE_PROXY_TARGET` in `web/frontend/.env.local` to a matching port (both files are gitignored).

On a fresh database the app opens a one-time setup wizard at `/welcome` to create the first admin account.

## Checks (must pass before a PR)

`make ci` runs the full gate. Individually:

```bash
go test ./... -race          # Go tests with the race detector
gofmt -s -l internal/ cmd/   # must print nothing
golangci-lint run --timeout=3m
( cd web/frontend && npx tsc -b && npm run build )
```

- **Go:** table-driven tests with parallel execution; handwritten mocks colocated in `_test.go` files. Every exported function gets a GoDoc comment. Wrap errors with `fmt.Errorf("context: %w", err)`.
- **Frontend:** `tsc -b` must pass (strict, no unused locals/params). No `any` unless unavoidable. All HTTP goes through `src/lib/api.ts`.

See `CLAUDE.md` for the full conventions.

## Pull requests

1. Branch off `main` (don't commit directly to `main`).
2. Keep the change focused; update `ARCHITECTURE.md` / `CLAUDE.md` / `README.md` when behavior or conventions change.
3. Add or update tests — every JSON endpoint needs at least a smoke test.
4. Make sure `make ci` is green.
5. Open a PR using the template; describe the change and how you verified it.

## Reporting issues

Use the issue templates (bug report / feature request). For security-sensitive reports, please avoid filing a public issue with exploit details — note that the runtime-MDX docs feature is admin-gated and see the security notes in `ARCHITECTURE.md`.
