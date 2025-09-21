# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o golinks ./cmd/server

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite tzdata

# Create non-root user
RUN addgroup -g 1001 -S golinks && \
    adduser -u 1001 -S golinks -G golinks

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/golinks .

# Copy web assets
COPY --from=builder /app/web ./web

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown -R golinks:golinks /app

# Switch to non-root user
USER golinks

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/homepage/ || exit 1

# Set environment variables
ENV PORT=8080
ENV DATABASE_PATH=/app/data/golinks.db
ENV BASE_URL=http://localhost:8080
ENV ENVIRONMENT=production

# Run the application
CMD ["./golinks"]
