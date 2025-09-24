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

	// Initialize services
	appLogger.Info("Initializing services")
	linkService := service.NewLinkService(shortcutRepo, queryRepo, appLogger)
	docService := service.NewDocumentService("docs", appLogger)

	// Initialize handlers
	appLogger.Info("Initializing handlers")
	handler := handlers.NewHandler(linkService, cfg, appLogger)
	docHandler := handlers.NewDocumentHandler(docService, appLogger)

	// Setup router
	appLogger.Info("Setting up HTTP router")
	router := mux.NewRouter()

	handler.RegisterRoutes(router)
	docHandler.RegisterRoutes(router)

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
