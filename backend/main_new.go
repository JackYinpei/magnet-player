package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/torrentplayer/backend/config"
	"github.com/torrentplayer/backend/db"
	"github.com/torrentplayer/backend/handlers"
	"github.com/torrentplayer/backend/middleware"
	"github.com/torrentplayer/backend/service"
	"github.com/torrentplayer/backend/torrent"
)

// Application represents the main application structure
type Application struct {
	config         *config.Config
	dbManager      *db.DatabaseManager
	torrentClient  *torrent.Client
	torrentStore   *db.TorrentStore
	torrentService *service.TorrentService
	searchService  *service.SearchService
	server         *http.Server
}

// NewApplication creates a new application instance with all dependencies
func NewApplication() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log.Printf("Starting Magnet Player Server (Environment: %s)", cfg.Server.Env)

	// Initialize database manager
	dbManager, err := db.NewDatabaseManager(
		cfg.Database.Path,
		cfg.Database.MaxConnections,
		time.Duration(cfg.Database.ConnMaxLifetime)*time.Second,
	)
	if err != nil {
		return nil, err
	}

	// Initialize torrent client
	torrentClient, err := torrent.NewClient(cfg.Torrent.DataDir)
	if err != nil {
		dbManager.Close()
		return nil, err
	}

	// Initialize torrent store
	torrentStore, err := db.NewTorrentStore(dbManager)
	if err != nil {
		torrentClient.Close()
		dbManager.Close()
		return nil, err
	}

	// Initialize services
	torrentService := service.NewTorrentService(torrentClient, torrentStore, cfg)
	searchService := service.NewSearchService(cfg)

	// Restore torrents from database
	if err := torrentService.RestoreTorrentsFromDB(); err != nil {
		log.Printf("Warning: Failed to restore torrents from database: %v", err)
	}

	app := &Application{
		config:         cfg,
		dbManager:      dbManager,
		torrentClient:  torrentClient,
		torrentStore:   torrentStore,
		torrentService: torrentService,
		searchService:  searchService,
	}

	// Setup HTTP server
	app.setupServer()

	return app, nil
}

// setupServer configures the HTTP server with middleware and routes
func (app *Application) setupServer() {
	// Create handlers
	torrentHandler := handlers.NewTorrentHandler(app.torrentService, app.searchService)
	streamHandler := handlers.NewStreamHandler(app.torrentService)
	searchHandler := handlers.NewSearchHandler(app.searchService)

	// Setup router with middleware
	mux := http.NewServeMux()

	// Apply middleware
	corsConfig := middleware.DefaultCORSConfig()
	if app.config.IsDevelopment() {
		// Allow all origins in development
		corsConfig.AllowedOrigins = []string{"*"}
	}

	// Create middleware chain
	chain := middleware.CORS(corsConfig)
	logger := middleware.Logger
	errorHandler := middleware.ErrorHandler

	// Register routes with middleware
	mux.HandleFunc("/magnet/api/magnet", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("POST", "OPTIONS")(
				middleware.ValidateJSONBody(1024*1024)(
					torrentHandler.AddMagnet))))).ServeHTTP)

	mux.HandleFunc("/magnet/api/torrents", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("GET", "OPTIONS")(
				torrentHandler.ListTorrents)))).ServeHTTP)

	mux.HandleFunc("/magnet/api/movie-details/", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("POST", "OPTIONS")(
				middleware.ValidateJSONBody(1024*1024)(
					torrentHandler.UpdateMovieDetails))))).ServeHTTP)

	mux.HandleFunc("/magnet/api/get-movie-details", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("GET", "OPTIONS")(
				torrentHandler.GetMovieDetails)))).ServeHTTP)

	mux.HandleFunc("/magnet/api/torrents/save-data/", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("POST", "OPTIONS")(
				middleware.ValidateJSONBody(2*1024*1024)(
					torrentHandler.SaveTorrentData))))).ServeHTTP)

	mux.HandleFunc("/magnet/stream/", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("GET", "OPTIONS")(
				streamHandler.StreamFile)))).ServeHTTP)

	mux.HandleFunc("/magnet/search", 
		chain(logger(errorHandler(
			middleware.ValidateMethod("GET", "OPTIONS")(
				middleware.ValidateQueryParams(map[string]bool{
					"filename": true,
				})(searchHandler.SearchMovie))))).ServeHTTP)

	// Setup server
	app.server = &http.Server{
		Addr:         app.config.GetServerAddress(),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// Start starts the application server
func (app *Application) Start() error {
	log.Printf("Server starting on %s", app.config.GetServerAddress())
	return app.server.ListenAndServe()
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Shutdown HTTP server
	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close torrent client
	if app.torrentClient != nil {
		log.Println("Closing torrent client...")
		app.torrentClient.Close()
	}

	// Close database connections
	if app.torrentStore != nil {
		log.Println("Closing torrent store...")
		app.torrentStore.Close()
	}

	if app.dbManager != nil {
		log.Println("Closing database manager...")
		// Optimize database before closing
		if err := app.dbManager.Optimize(); err != nil {
			log.Printf("Database optimization failed: %v", err)
		}
		app.dbManager.Close()
	}

	log.Println("Shutdown complete")
	return nil
}

// main is the application entry point
func main() {
	// Create application
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := app.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-quit
	log.Println("Received shutdown signal")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform graceful shutdown
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Forced shutdown: %v", err)
		os.Exit(1)
	}

	log.Println("Server stopped")
}