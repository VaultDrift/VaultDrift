package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/federation"
	"github.com/vaultdrift/vaultdrift/internal/media"
	"github.com/vaultdrift/vaultdrift/internal/preview"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
	"github.com/vaultdrift/vaultdrift/web"
)

// Server is the HTTP server for VaultDrift.
type Server struct {
	httpServer *http.Server
	router     *http.ServeMux
	db         *db.Manager
	authSvc    *auth.Service
	vfs        *vfs.VFS
	storage    storage.Backend
	config     config.ServerConfig
	jwtSecret  []byte
	rbac       *auth.RBAC
	events     *EventNotifier
	federation *federation.Manager
	metrics    *Metrics
}

// NewServer creates a new HTTP server.
func NewServer(cfg config.ServerConfig, database *db.Manager, authService *auth.Service, vfsService *vfs.VFS, store storage.Backend, jwtSecret []byte, fed *federation.Manager) *Server {
	router := http.NewServeMux()

	// Create metrics collector
	metrics := NewMetrics(database)

	s := &Server{
		router:     router,
		db:         database,
		authSvc:    authService,
		vfs:        vfsService,
		storage:    store,
		config:     cfg,
		jwtSecret:  jwtSecret,
		rbac:       auth.NewRBAC(database),
		events:     NewEventNotifier(vfsService, database),
		federation: fed,
		metrics:    metrics,
	}

	// Setup routes
	s.setupRoutes()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.wrapMiddleware(router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// Start starts the server and blocks until shutdown.
func (s *Server) Start() error {
	// Setup graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		s.httpServer.SetKeepAlivesEnabled(false)
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Printf("Server is ready to handle requests at %s:%d\n", s.config.Host, s.config.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s:%d: %w", s.config.Host, s.config.Port, err)
	}

	<-done
	log.Println("Server stopped")
	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /ready", s.handleReady)

	// Metrics endpoints (public for monitoring, should be protected by reverse proxy)
	s.router.Handle("GET /metrics/json", s.metrics.MetricsHandler())
	s.router.Handle("GET /metrics/prometheus", s.metrics.PrometheusHandler())
	s.router.Handle("GET /metrics", s.metrics.PrometheusHandler()) // Default to Prometheus format

	// Auth handlers (public)
	authHandler := NewAuthHandler(s.authSvc)
	authHandler.RegisterRoutes(s.router)

	// Create auth middleware
	authMiddleware := NewAuthMiddleware(s.authSvc, nil, s.rbac, s.jwtSecret)

	// User handlers (profile, settings)
	userHandler := NewUserHandler(s.db, s.authSvc)
	userHandler.RegisterRoutes(s.router, authMiddleware)

	// File handlers
	fileHandler := NewFileHandler(s.vfs, s.db, s.events)
	fileHandler.RegisterRoutes(s.router, authMiddleware)

	// Folder handlers
	folderHandler := NewFolderHandler(s.vfs, s.events)
	folderHandler.RegisterRoutes(s.router, authMiddleware)

	// Upload handlers
	uploadHandler := NewUploadHandler(s.vfs, s.db, s.storage)
	uploadHandler.RegisterRoutes(s.router, authMiddleware)

	// Download handlers
	downloadHandler := NewDownloadHandler(s.vfs, s.db, s.storage)
	downloadHandler.RegisterRoutes(s.router, authMiddleware)

	// Share handlers (authenticated)
	shareHandler := NewShareHandler(s.vfs, s.db, s.events)
	shareHandler.RegisterRoutes(s.router, authMiddleware)

	// Public share handlers (no auth required)
	publicShareHandler := NewPublicShareHandler(s.db, s.storage)
	publicShareHandler.RegisterRoutes(s.router)

	// Trash handlers
	trashHandler := NewTrashHandler(s.vfs, s.db)
	trashHandler.RegisterRoutes(s.router, authMiddleware)

	// Version handlers
	versionHandler := NewVersionHandler(s.vfs)
	versionHandler.RegisterRoutes(s.router, authMiddleware)

	// Media streaming handlers (HLS transcoding)
	mediaHandler := media.NewStreamHandler(s.vfs, s.db, s.storage)
	mediaHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)

	// Document preview handlers
	previewHandler := preview.NewHandler(s.vfs, s.db, s.storage)
	previewHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)

	// Thumbnail handlers
	thumbCacheDir := filepath.Join("data", "thumbnails")
	thumbnailHandler := NewThumbnailHandler(s.vfs, s.db, s.storage, thumbCacheDir)
	thumbnailHandler.RegisterRoutes(s.router, authMiddleware)

	// Real-time event streaming (SSE)
	s.events.RegisterRoutes(s.router, authMiddleware)

	// Federation handlers
	if s.federation != nil && s.federation.IsEnabled() {
		fedHandler := NewFederationHandler(s.federation)
		fedHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)
	}

	// Static files (for web UI) - embedded from web/dist
	webFS := web.FS()
	s.router.Handle("/", http.FileServer(webFS))
}

// wrapMiddleware applies global middleware.
func (s *Server) wrapMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (last applied is first executed)
	handler = RecoveryMiddleware(handler)

	// Add rate limiting if enabled
	if s.config.RateLimit.Enabled {
		rateLimiter := NewRateLimitMiddleware(
			s.config.RateLimit.Requests,
			s.config.RateLimit.Window,
		)
		handler = rateLimiter.Limit(handler)
	}

	handler = LoggingMiddleware(handler)
	handler = CORSMiddleware(handler, nil)
	handler = SecurityHeadersMiddleware(handler)
	return handler
}

// handleHealth returns detailed health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	health := map[string]interface{}{
		"status":    "healthy",
		"version":   "0.1.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"system": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]uint64{
				"alloc":       memStats.Alloc,
				"total_alloc": memStats.TotalAlloc,
				"sys":         memStats.Sys,
				"heap_alloc":  memStats.HeapAlloc,
				"heap_inuse":  memStats.HeapInuse,
			},
			"gc_count": memStats.NumGC,
		},
		"dependencies": map[string]string{
			"database":   "ok",
			"storage":    "ok",
			"web_socket": "ok",
		},
	}

	// Check database
	if err := s.db.Ping(r.Context()); err != nil {
		health["status"] = "degraded"
		deps := health["dependencies"].(map[string]string)
		deps["database"] = "error: " + err.Error()
	}

	writeJSON(w, http.StatusOK, health)
}

// handleReady returns readiness status.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  "database unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
