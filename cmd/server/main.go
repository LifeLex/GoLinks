package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golinks/internal/config"
	"golinks/internal/database"
	"golinks/internal/handlers"
	"golinks/internal/logger"
	"golinks/internal/repository"
	"golinks/internal/service"
	"golinks/web/frontend"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize simple logging
	logger.Initialize(cfg.Logging)
	appLogger := logger.Default()

	appLogger.Info("Starting GoLinks application on port %d (env: %s)", cfg.Port, cfg.Environment)

	// Initialize database
	appLogger.Info("Initializing database: %s", cfg.DatabasePath)
	db, err := database.NewSQLiteDB(cfg.DatabasePath)
	if err != nil {
		appLogger.Error("Failed to initialize database: %v", err)
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	appLogger.Info("Running database migrations")
	if err := database.Migrate(db); err != nil {
		appLogger.Error("Failed to run migrations: %v", err)
		log.Fatalf("Failed to run migrations: %v", err)
	}
	appLogger.Info("Database migrations completed successfully")

	// Initialize repositories
	appLogger.Info("Initializing repositories")
	shortcutRepo := repository.NewShortcutRepository(db, appLogger)
	queryRepo := repository.NewQueryRepository(db, appLogger)
	userRepo := repository.NewUserRepository(db, appLogger)
	sessionRepo := repository.NewSessionRepository(db, appLogger)

	// Initialize services
	appLogger.Info("Initializing services")
	linkService := service.NewLinkService(shortcutRepo, queryRepo, appLogger)
	docService := service.NewDocumentService("docs", appLogger)
	authService := service.NewAuthService(
		userRepo, sessionRepo, appLogger,
		time.Duration(cfg.Auth.SessionTTLHours)*time.Hour,
		cfg.Auth.BcryptCost, cfg.Auth.MinPasswordLen,
	)

	// Initialize handlers
	appLogger.Info("Initializing handlers")
	handler := handlers.NewHandler(linkService, cfg, appLogger)
	docHandler := handlers.NewDocumentHandler(docService, appLogger)
	authHandler := handlers.NewAuthHandler(authService, cfg, appLogger)
	authMiddleware := handlers.NewAuthMiddleware(authService, appLogger)

	// Setup router. A global Authenticate middleware loads the optional user
	// into every request's context; per-route gating is done via subrouters.
	appLogger.Info("Setting up HTTP router")
	router := mux.NewRouter()
	router.Use(authMiddleware.Authenticate)

	// Subrouters carry their own middleware. Routes registered on `authed`
	// require a logged-in user; routes on `admin` require the admin role.
	// Method-specific routes (GET on public, POST on authed for the same path)
	// coexist because mux matches by method.
	authed := router.NewRoute().Subrouter()
	authed.Use(authMiddleware.RequireAuth)
	admin := router.NewRoute().Subrouter()
	admin.Use(authMiddleware.RequireAdmin)

	handler.RegisterRoutes(router, authed)
	docHandler.RegisterRoutes(router, admin)
	authHandler.RegisterRoutes(router, admin)

	// Catch-all: serve the embedded SPA for every unmatched GET.
	// Reserved prefixes guarantee we never shadow API/redirect/auth routes even
	// if the router's match order changes.
	router.PathPrefix("/").Handler(frontend.Handler("api", "query", "auth"))

	// Periodically purge expired sessions.
	stopCleanup := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n, err := authService.PurgeExpiredSessions(context.Background()); err != nil {
					appLogger.Error("Session cleanup failed: %v", err)
				} else if n > 0 {
					appLogger.Info("Purged %d expired sessions", n)
				}
			case <-stopCleanup:
				return
			}
		}
	}()
	defer close(stopCleanup)

	// Setup server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info("Starting HTTP server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed to start: %v", err)
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Received shutdown signal, initiating graceful shutdown")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appLogger.Info("Shutting down HTTP server (timeout: 30s)")
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown: %v", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server shutdown completed successfully")
}
