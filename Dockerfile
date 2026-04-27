# syntax=docker/dockerfile:1.6

# -----------------------------------------------------------------------------
# Stage 1: build the React/Vite frontend into web/frontend/dist
# -----------------------------------------------------------------------------
FROM node:20-alpine AS frontend
WORKDIR /app/web/frontend
COPY web/frontend/package*.json ./
RUN npm ci
COPY web/frontend/ ./
RUN npm run build

# -----------------------------------------------------------------------------
# Stage 2: compile the Go binary with the built frontend embedded
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS backend
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Drop any pre-existing dist (in case the repo's stub is present) and copy the
# freshly-built assets over the top so go:embed picks them up.
RUN rm -rf web/frontend/dist
COPY --from=frontend /app/web/frontend/dist ./web/frontend/dist

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o golinks ./cmd/server

# -----------------------------------------------------------------------------
# Stage 3: minimal runtime with the single binary only — no web/ directory.
# -----------------------------------------------------------------------------
FROM alpine:3.18
RUN apk add --no-cache ca-certificates sqlite tzdata

RUN addgroup -g 1001 -S golinks && \
    adduser -u 1001 -S golinks -G golinks

WORKDIR /app
COPY --from=backend /app/golinks .
# docs/ is still read from disk because uploads are persisted there.
COPY --from=backend /app/docs ./docs
RUN mkdir -p /app/data && chown -R golinks:golinks /app

USER golinks

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

ENV PORT=8080
ENV DATABASE_PATH=/app/data/golinks.db
ENV BASE_URL=http://localhost:8080
ENV ENVIRONMENT=production

CMD ["./golinks"]
