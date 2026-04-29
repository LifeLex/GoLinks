// Command server is the GoLinks HTTP server. It is the composition root: every
// adapter and use case is wired up here, and nowhere else.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golinks/internal/adapters/filesystem"
	"golinks/internal/adapters/httpapi"
	"golinks/internal/adapters/persistence"
	_ "golinks/internal/adapters/persistence/migrations" // registers Goose migrations via init()
	"golinks/internal/core/docs"
	"golinks/internal/core/links"
	"golinks/internal/platform/config"
	"golinks/internal/platform/database"
	"golinks/internal/platform/logger"
	"golinks/web/frontend"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logger.
	logger.Initialize(cfg.Logging)
	appLogger := logger.Default()

	appLogger.Info("Starting GoLinks application on port %d (env: %s)", cfg.Port, cfg.Environment)

	// Open database connection. Driver and URL come from config; sqlite is
	// the default, postgres is selected via DATABASE_DRIVER=postgres.
	driver := database.Driver(cfg.DatabaseDriver)
	appLogger.Info("Initializing database: driver=%s url=%s", driver, redactedURL(cfg.DatabaseURL))
	db, err := database.OpenGorm(driver, cfg.DatabaseURL)
	if err != nil {
		appLogger.Error("Failed to initialize database: %v", err)
		log.Fatalf("Failed to initialize database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("Failed to access underlying *sql.DB: %v", err)
		log.Fatalf("Failed to access underlying *sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// Run migrations via Goose.
	appLogger.Info("Running database migrations")
	if err := database.Migrate(db, driver); err != nil {
		appLogger.Error("Failed to run migrations: %v", err)
		log.Fatalf("Failed to run migrations: %v", err)
	}
	appLogger.Info("Database migrations completed successfully")

	// Outbound adapters.
	appLogger.Info("Initializing outbound adapters")
	linksRepo := persistence.NewLinksRepo(db, appLogger)
	queriesRepo := persistence.NewQueriesRepo(db, appLogger)
	docsStore := filesystem.NewDocStore("docs")

	// Use cases.
	appLogger.Info("Initializing use cases")
	linkService := links.NewService(linksRepo, queriesRepo, appLogger)
	docService := docs.NewService(docsStore, appLogger)

	// Inbound adapters.
	appLogger.Info("Initializing inbound adapters")
	linksHandler := httpapi.NewLinksHandler(linkService, cfg, appLogger)
	docsHandler := httpapi.NewDocsHandler(docService, appLogger)

	// Router.
	appLogger.Info("Setting up HTTP router")
	router := mux.NewRouter()
	linksHandler.Register(router)
	docsHandler.Register(router)
	// SPA catch-all (last). Reserved prefixes guarantee /api/* and /query/*
	// never accidentally fall through to the frontend handler.
	router.PathPrefix("/").Handler(frontend.Handler("api", "query"))

	// Server with graceful shutdown.
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLogger.Info("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed to start: %v", err)
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Received shutdown signal, initiating graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appLogger.Info("Shutting down HTTP server (timeout: 30s)")
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown: %v", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server shutdown completed successfully")
}

// redactedURL strips credentials from a database URL for safe logging. SQLite
// paths pass through unchanged; postgres URLs have any password in the
// userinfo portion replaced with "***".
func redactedURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := neturl.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	if _, hasPassword := u.User.Password(); !hasPassword {
		return raw
	}
	u.User = neturl.UserPassword(u.User.Username(), "***")
	return u.String()
}
