# GoLinks

A modern, minimalist URL shortener inspired by Google's internal golinks system. Built with Go, HTMX, and a Dieter Rams-inspired design philosophy.

## Features

- **Simple URL Shortening**: Create memorable shortcuts for long URLs
- **Variable Substitution**: Use `{*}` placeholders for dynamic content
- **Usage Analytics**: Track popular queries and usage patterns
- **Clean Architecture**: Modular, testable, and maintainable codebase
- **Modern UI**: HTMX-powered interface with Dieter Rams-inspired design
- **Containerized**: Ready-to-deploy Docker container

## Quick Start

### Using Docker (Recommended)

```bash
# Clone the repository
git clone <repository-url>
cd golinks

# Start the application
docker-compose up -d

# Access the application
open http://localhost:8080/homepage/
```

### Local Development

```bash
# Install dependencies
make deps

# Copy environment configuration
cp env.example .env

# Run the application
make run

# Or run with hot reload (requires air)
make dev

# Access the application
open http://localhost:8080/homepage/
```

## Browser Setup

To use GoLinks, configure your browser to use the service as a search engine:

1. **Chrome/Edge**: Settings → Search engine → Manage search engines → Add
   - Search engine: `GoLinks`
   - Shortcut: `go`
   - URL: `http://localhost:8080/query/%s`

2. **Firefox**: Bookmarks → Add bookmark
   - Name: `GoLinks`
   - Location: `http://localhost:8080/query/%s`
   - Keyword: `go`

3. Visit `/setup/` for detailed setup instructions

## Usage Examples

After setup, you can use GoLinks directly from your browser's address bar:

```
go docs                    # Navigate to documentation
go jira 123               # Open JIRA ticket 123 (if configured with {*})
go github myproject       # Search GitHub for "myproject"
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `BASE_URL` | `http://localhost:8080` | Base URL for the service |
| `ENVIRONMENT` | `development` | Environment (development/production) |
| `LOG_LEVEL` | `info` | Logger level (debug/info/warn/error) |
| `DATABASE_DRIVER` | `sqlite` | Storage backend — `sqlite` or `postgres` |
| `DATABASE_URL` | derived from `DATABASE_PATH` | Connection string. SQLite: a path or `file:` DSN. Postgres: `postgres://user:pass@host:5432/db?sslmode=disable` |
| `DATABASE_PATH` | `golinks.db` | Deprecated; only used to synthesise `DATABASE_URL` for SQLite when `DATABASE_URL` is empty |

See `.env.example` for a copy-pasteable template.

### Running with Postgres

```bash
docker compose up --build
```

`docker-compose.yml` boots GoLinks against a local Postgres container. Migrations run automatically on first start.

### Creating Links

1. Visit the homepage at `/homepage/`
2. Use the "Add new keyword" form
3. Enter a keyword and target URL
4. Use `{*}` in URLs for variable substitution

### Variable Substitution

GoLinks supports dynamic URLs using `{*}` placeholders:

```
Keyword: github
URL: https://github.com/search?q={*}
Usage: go github awesome-project
Result: https://github.com/search?q=awesome-project
```

## Architecture

Hexagonal architecture: a feature-led `core/` of business logic with explicit ports, surrounded by `adapters/` that implement them, and a `platform/` layer for cross-cutting infrastructure. See `ARCHITECTURE.md` for the full reference.

```
cmd/server/                      # Composition root
internal/
├── core/                        # Business logic
│   ├── links/                   # entity + ports + service
│   └── docs/                    # entity + ports + service
├── adapters/
│   ├── httpapi/                 # Inbound: HTTP handlers (JSON + redirect)
│   ├── persistence/             # Outbound: GORM repositories
│   └── filesystem/              # Outbound: docs/ folder adapter
└── platform/
    ├── config/                  # Env loading
    ├── logger/                  # slog wrapper
    └── database/                # GORM connection + Goose migrations
web/frontend/                    # React + Vite SPA (embedded into the binary)
docs/                            # User-uploaded markdown/MDX
```

## Design Philosophy

The UI follows Dieter Rams' principles of good design:

- **Innovative**: Modern web technologies (HTMX, CSS Grid)
- **Useful**: Focused on core functionality without bloat
- **Aesthetic**: Clean, minimal interface with purposeful typography
- **Understandable**: Clear information hierarchy and intuitive navigation
- **Unobtrusive**: Subtle interactions that don't distract
- **Honest**: Transparent about functionality and limitations
- **Long-lasting**: Timeless design that won't feel outdated
- **Thorough**: Attention to detail in every interaction
- **Environmentally friendly**: Efficient, lightweight implementation
- **Minimal**: Only essential elements, nothing superfluous

## Development

### Project Structure

- **Domain Layer**: Core business logic and entities
- **Service Layer**: Use cases and business rules
- **Repository Layer**: Data access and persistence
- **Handler Layer**: HTTP transport and presentation
- **Infrastructure**: Database and external services

### Development Commands

```bash
# Format and check code
make fmt

# Fix formatting and linting issues
make fix

# Run linter
make lint

# Run tests with coverage
make test

# Build binary
make build

# Build Docker image
make docker-build

# Run all CI checks
make ci

# Clean build artifacts
make clean
```

## Deployment

### Docker

```bash
# Production deployment
docker-compose up -d

# Or using make commands
make docker-build
make docker-run
```

### Environment Variables for Production

```bash
export PORT=8080
export BASE_URL=https://go.yourcompany.com
export ENVIRONMENT=production

# SQLite (default) — needs a writable persistent volume
export DATABASE_DRIVER=sqlite
export DATABASE_URL=/data/golinks.db

# OR: Postgres
export DATABASE_DRIVER=postgres
export DATABASE_URL="postgres://user:pass@db:5432/golinks?sslmode=require"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Acknowledgments

- Inspired by Google's internal golinks system
- Design philosophy based on Dieter Rams' principles
- Built with modern Go best practices and Clean Architecture
