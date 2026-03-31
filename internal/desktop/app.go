package desktop

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/server"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// App represents the desktop application
type App struct {
	config  *config.Config
	server  *server.Server
	vfs     *vfs.VFS
	db      *db.Manager
	storage storage.Backend
	ctx     context.Context
	cancel  context.CancelFunc
	tray    *TrayMenu
}

// NewApp creates a new desktop application
func NewApp(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize database
	database, err := db.Open(db.Config{Path: cfg.Database.Path})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize storage backend
	store, err := storage.NewBackend(cfg.Storage)
	if err != nil {
		database.Close()
		cancel()
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize VFS
	vfsService := vfs.NewVFS(database)

	// Initialize auth service
	authSvc := auth.NewService(database, []byte(cfg.Auth.JWTSecret))

	// Create HTTP server (federation disabled for desktop)
	httpServer := server.NewServer(cfg.Server, database, authSvc, vfsService, store, []byte(cfg.Auth.JWTSecret), nil)

	// Create app
	app := &App{
		config:  cfg,
		server:  httpServer,
		vfs:     vfsService,
		db:      database,
		storage: store,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create tray menu
	app.tray = NewTrayMenu(app)

	return app, nil
}

// Run starts the desktop application
func (a *App) Run() error {
	defer a.cleanup()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		log.Println("Starting VaultDrift server...")
		if err := a.server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for server to be ready
	// In production, we'd check the health endpoint
	// For now, just give it a moment to start
	select {
	case err := <-serverErr:
		return fmt.Errorf("server failed to start: %w", err)
	case <-sigChan:
		return nil
	default:
		// Continue to UI
	}

	// Run the UI
	uiErr := make(chan error, 1)
	go func() {
		uiErr <- a.runUI()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-serverErr:
		return err
	case err := <-uiErr:
		return err
	case <-sigChan:
		log.Println("Shutting down...")
		return nil
	}
}

// runUI runs the desktop UI
func (a *App) runUI() error {
	log.Printf("VaultDrift desktop app running at http://localhost:%d", a.config.Server.Port)
	log.Println("Use the system tray icon to access the application")

	// Run the system tray menu
	return a.tray.Run()
}

// cleanup releases resources
func (a *App) cleanup() {
	a.cancel()

	if a.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		a.server.Stop(ctx)
	}

	if a.db != nil {
		a.db.Close()
	}
}

// IsServerRunning checks if the server is accessible
func (a *App) IsServerRunning() bool {
	// TODO: Implement health check
	return true
}
