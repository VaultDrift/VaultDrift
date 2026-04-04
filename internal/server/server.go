package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/federation"
	"github.com/vaultdrift/vaultdrift/internal/media"
	"github.com/vaultdrift/vaultdrift/internal/preview"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	syncpkg "github.com/vaultdrift/vaultdrift/internal/sync"
	"github.com/vaultdrift/vaultdrift/internal/tracing"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
	"github.com/vaultdrift/vaultdrift/internal/webdav"
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
	events        *EventNotifier
	federation    *federation.Manager
	metrics       *Metrics
	uploadHandler *UploadHandler
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
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Start starts the server and blocks until ListenAndServe returns.
// Use Stop() for graceful shutdown from external signal handlers.
func (s *Server) Start() error {
	log.Printf("Server is ready to handle requests at %s:%d\n", s.config.Host, s.config.Port)

	var listenErr error
	if s.config.TLS.Enabled {
		certFile := s.config.TLS.CertFile
		keyFile := s.config.TLS.KeyFile
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("TLS enabled but cert_file or key_file not configured")
		}
		listenErr = s.httpServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		listenErr = s.httpServer.ListenAndServe()
	}
	if listenErr != nil && listenErr != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s:%d: %w", s.config.Host, s.config.Port, listenErr)
	}

	log.Println("Server stopped")
	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop(ctx context.Context) error {
	// Stop background goroutines
	if s.events != nil {
		s.events.Close()
	}
	if s.uploadHandler != nil {
		s.uploadHandler.Close()
	}
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /ready", s.handleReady)

	// Auth handlers (public)
	authHandler := NewAuthHandler(s.authSvc)
	authHandler.RegisterRoutes(s.router)

	// Create auth middleware
	authMiddleware := NewAuthMiddleware(s.authSvc, nil, s.rbac, s.jwtSecret)

	// Metrics endpoints (require authentication)
	s.router.Handle("GET /metrics/json", authMiddleware.Authenticate(s.metrics.MetricsHandler()))
	s.router.Handle("GET /metrics/prometheus", authMiddleware.Authenticate(s.metrics.PrometheusHandler()))
	s.router.Handle("GET /metrics", authMiddleware.Authenticate(s.metrics.PrometheusHandler()))

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
	s.uploadHandler = NewUploadHandler(s.vfs, s.db, s.storage)
	s.uploadHandler.RegisterRoutes(s.router, authMiddleware)

	// Download handlers
	downloadHandler := NewDownloadHandler(s.vfs, s.db, s.storage)
	downloadHandler.RegisterRoutes(s.router, authMiddleware)

	// Share handlers (authenticated)
	shareHandler := NewShareHandler(s.vfs, s.db, s.events, s.config.Sharing)
	shareHandler.RegisterRoutes(s.router, authMiddleware)

	// Public share handlers (no auth required, rate-limited)
	publicShareHandler := NewPublicShareHandler(s.db, s.storage)
	publicShareRL := NewRateLimitMiddleware(60, time.Minute)
	publicShareHandler.RegisterRoutes(s.router, publicShareRL.Limit)

	// Trash handlers
	trashHandler := NewTrashHandler(s.vfs, s.db)
	trashHandler.RegisterRoutes(s.router, authMiddleware)

	// Version handlers
	versionHandler := NewVersionHandler(s.vfs, s.db)
	versionHandler.RegisterRoutes(s.router, authMiddleware)

	// Admin handlers (system stats, user management, audit logs, profiling)
	adminHandler := NewAdminHandler(s.db, s.authSvc, s.storage, nil)
	adminHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)

	// Backup handlers (database backup/restore)
	backupHandler := NewBackupHandler(s.db, "data")
	backupHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)

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

	// WebSocket real-time updates
	wsServer := NewWebSocketServer(s.vfs, s.db, s.jwtSecret, s.config.AllowedOrigins)
	wsServer.RegisterRoutes(s.router, authHandler)

	// Sync handlers (device management, negotiate, push, pull, commit)
	syncEngine := syncpkg.NewEngine(s.db, s.storage)
	syncHandler := NewSyncHandler(syncEngine, s.db, s.storage)
	syncHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)

	// Federation handlers
	if s.federation != nil && s.federation.IsEnabled() {
		fedHandler := NewFederationHandler(s.federation)
		fedHandler.RegisterRoutes(s.router, authMiddleware.Authenticate)
	}

	// Static files (for web UI) - embedded from web/dist
	webFS := web.FS()
	s.router.Handle("/", http.FileServer(webFS))

	// WebDAV handler (mounted on /webdav/)
	webdavHandler := webdav.NewHandler(s.vfs, s.db, s.storage, "/webdav")
		webdavHandler.SetCredentialValidator(func(ctx context.Context, username, password string) (string, bool) {
			user, err := s.db.GetUserByUsername(ctx, username)
			if err != nil {
				return "", false
			}
			if user.Status != "active" {
				return "", false
			}
			if user.TOTPEnabled {
				// WebDAV Basic Auth cannot handle 2FA; reject
				return "", false
			}
			ok, err := auth.VerifyPassword(password, user.PasswordHash)
			if err != nil || !ok {
				return "", false
			}
			return user.ID, true
		})
	s.router.Handle("/webdav/", authMiddleware.Authenticate(webdavHandler))
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
	handler = CORSMiddleware(handler, s.config.AllowedOrigins)
	handler = SecurityHeadersMiddleware(handler)

	// Tracing middleware (no-op if no provider initialized)
	tracingMW := tracing.NewHTTPMiddleware()
	handler = tracingMW.Middleware(handler)

	return handler
}

// handleHealth returns detailed health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	deps := map[string]string{
		"database":   "ok",
		"storage":    "ok",
		"web_socket": "ok",
	}

	status := "healthy"

	// Check database connectivity
	if err := s.db.Ping(r.Context()); err != nil {
		status = "degraded"
		deps["database"] = "unavailable"
	}

	// Check storage backend connectivity
	if _, err := s.storage.Stats(r.Context()); err != nil {
		status = "degraded"
		deps["storage"] = "unavailable"
	}

	health := map[string]interface{}{
		"status":    status,
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
		"dependencies": deps,
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
