# GoLinks

A modern, minimalist URL shortener inspired by Google's internal golinks system. Built with Go, HTMX, and a Dieter Rams-inspired design philosophy.

## Features

- **Simple URL Shortening**: Create memorable shortcuts for long URLs
- **Variable Substitution**: Use `{*}` placeholders for dynamic content
- **Recursive Aliases**: Keywords can point to other keywords
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
| `DATABASE_PATH` | `golinks.db` | SQLite database path |
| `BASE_URL` | `http://localhost:8080` | Base URL for the service |
| `ENVIRONMENT` | `development` | Environment (development/production) |

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

The application follows Clean Architecture principles:

```
cmd/server/          # Application entrypoint
internal/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── domain/          # Domain models and interfaces
├── handlers/        # HTTP handlers and routing
├── repository/      # Data access layer
└── service/         # Business logic layer
web/
├── static/          # CSS, images, and static assets
└── templates/       # HTML templates
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
export DATABASE_PATH=/data/golinks.db
export BASE_URL=https://go.yourcompany.com
export ENVIRONMENT=production
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
