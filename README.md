# GoLinks

A self-hosted URL shortener inspired by Google's internal `go/` links. Type `go jira 123` in your address bar and land on the right ticket.

One Go binary serves the redirect, a JSON API, and an embedded React SPA — no separate frontend service to deploy.

## Features

- Server-side `302` redirects (works as a browser search engine — no JavaScript required for the resolver)
- `{*}` placeholders for parameterised links (`go github {*}` → `github.com/search?q={*}`)
- Aliasing: a link's target can itself be a keyword (resolved recursively)
- Markdown / MDX documents served from `docs/`, compiled in the browser
- Single-binary distribution with the React build embedded via `//go:embed`

## Requirements

- Go 1.21+
- Node 20+ (for building the SPA)
- CGO toolchain (for the SQLite driver)

## Quick Start

```bash
git clone <repository-url>
cd golinks

cp env.example .env
make build         # builds the SPA, then the Go binary into ./build/golinks
./build/golinks
```

Open <http://localhost:8080>.

### Docker

```bash
docker compose up -d
```

The image is a three-stage build (node → go → alpine) producing a ~14 MB final image with no `web/` directory at runtime.

## Browser Setup

Configure GoLinks as a custom search engine pointed at `http://localhost:8080/query/%s` and assign it a keyword (e.g. `go`). Step-by-step instructions per browser are at `/setup` in the running app.

## Usage

```
go docs               # navigates to a link with the keyword "docs"
go jira 123           # link "jira" with URL "...?id={*}" → expands {*} → "123"
go google cats        # link "google" not literal — falls back to "google" with "cats" as the search term
```

## Configuration

All variables are optional. Defaults shown.

| Variable        | Default                   | Purpose                                       |
| --------------- | ------------------------- | --------------------------------------------- |
| `PORT`          | `8080`                    | HTTP listener port                            |
| `DATABASE_PATH` | `golinks.db`              | SQLite file path                              |
| `BASE_URL`      | `http://localhost:8080`   | Public base URL (returned to the SPA)         |
| `ENVIRONMENT`   | `development`             | Logged on startup; flips cookie `Secure` default |
| `LOG_LEVEL`     | `info`                    | `debug`, `info`, `warn`, `error`              |
| `SESSION_TTL_HOURS` | `720`                 | Login session lifetime (30 days)              |
| `COOKIE_SECURE` | prod: `true` / dev: `false` | HTTPS-only session cookie                   |
| `BCRYPT_COST`   | `12`                      | Password hashing work factor                  |
| `MIN_PASSWORD_LEN` | `8`                    | Minimum password length                       |

`.env` at the repo root is auto-loaded via `godotenv`.

## Accounts & access

GoLinks uses email + password authentication with server-side sessions (an `HttpOnly` cookie; only a hash of the session token is stored).

- **First run:** on a fresh database the app shows a one-time setup wizard at `/welcome`. The first account you create becomes the **admin**.
- **Adding users:** registration is closed after the first user. Admins add accounts from **Users** (`/admin/users`), choosing the `admin` or `user` role.
- **What's public vs. gated:** resolving golinks (`/query/...`) and browsing/searching the index stay public. Creating, editing, or deleting links requires signing in; uploading or deleting docs requires an admin (runtime MDX runs in the browser, so this is locked down).

For how sessions and the auth cookie work under the hood, see [`AUTH.md`](./AUTH.md).

## Development

```bash
make dev              # Go server (with air if installed) + Vite dev server concurrently
make frontend-dev     # Vite only, proxies /api and /query to :8080
make test             # go test ./... -race
make fmt              # gofmt + goimports
make lint             # golangci-lint
make ci               # full pipeline: frontend install + build, lint, test, build
```

The Vite dev server runs on `:5173` and proxies `/api/*` and `/query/*` to the Go backend on `:8080`, so the SPA can call relative URLs.

## Project Structure

```
cmd/server/                  Application entrypoint
internal/
├── config/                  Env / .env configuration
├── database/                SQLite connection + migrations
├── domain/                  Models (json + db tagged)
├── handlers/                HTTP handlers (JSON API + redirect)
├── logger/                  slog wrapper
├── repository/              Data access (database/sql, no ORM)
└── service/                 Business logic
web/frontend/                React 18 + TS + Vite SPA
└── dist/                    Build output, embedded via //go:embed all:dist
docs/                        Markdown / MDX served at runtime
```

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for the wiring diagram, request flow, and endpoint reference. See [`CLAUDE.md`](./CLAUDE.md) for contributor conventions.

## Acknowledgments

- Inspired by Google's internal `go/` links system
- Visual design tokens drawn from Dieter Rams' principles — Braun orange remains the primary accent
- Built with idiomatic Go and Clean Architecture

## License

See repository root.
