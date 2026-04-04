package server

import (
	"net/http"
	"net/http/pprof"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/worker"
)

// AdminHandler handles administrative operations.
type AdminHandler struct {
	db        *db.Manager
	authSvc   *auth.Service
	storage   storage.Backend
	workers   *worker.Manager
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(database *db.Manager, authService *auth.Service, store storage.Backend, workers *worker.Manager) *AdminHandler {
	return &AdminHandler{
		db:      database,
		authSvc: authService,
		storage: store,
		workers: workers,
	}
}

// RegisterRoutes registers admin routes (admin only).
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// System health
	mux.Handle("GET /api/v1/admin/health", auth(http.HandlerFunc(h.getSystemHealth)))

	// System stats
	mux.Handle("GET /api/v1/admin/stats", auth(http.HandlerFunc(h.getSystemStats)))

	// User management
	mux.Handle("GET /api/v1/admin/users", auth(http.HandlerFunc(h.listUsers)))
	mux.Handle("POST /api/v1/admin/users", auth(http.HandlerFunc(h.createUser)))
	mux.Handle("GET /api/v1/admin/users/{userID}", auth(http.HandlerFunc(h.getUser)))
	mux.Handle("PUT /api/v1/admin/users/{userID}", auth(http.HandlerFunc(h.updateUser)))
	mux.Handle("DELETE /api/v1/admin/users/{userID}", auth(http.HandlerFunc(h.deleteUser)))

	// Audit logs
	mux.Handle("GET /api/v1/admin/audit", auth(http.HandlerFunc(h.getAuditLogs)))

	// System maintenance
	mux.Handle("POST /api/v1/admin/maintenance/gc", auth(http.HandlerFunc(h.runGC)))
	mux.Handle("POST /api/v1/admin/maintenance/cleanup", auth(http.HandlerFunc(h.runCleanup)))

	// Profiling endpoints (admin only)
	h.registerPprofRoutes(mux, auth)
}

// requireAdmin wraps an http.HandlerFunc with an admin role check.
func (h *AdminHandler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.isAdmin(r) {
			ErrorResponse(w, http.StatusForbidden, "Admin access required")
			return
		}
		next(w, r)
	}
}

// registerPprofRoutes registers pprof profiling endpoints.
func (h *AdminHandler) registerPprofRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// CPU profile
	mux.Handle("GET /api/v1/admin/debug/pprof/", auth(http.HandlerFunc(h.requireAdmin(pprof.Index))))
	mux.Handle("GET /api/v1/admin/debug/pprof/cmdline", auth(http.HandlerFunc(h.requireAdmin(pprof.Cmdline))))
	mux.Handle("GET /api/v1/admin/debug/pprof/profile", auth(http.HandlerFunc(h.requireAdmin(pprof.Profile))))
	mux.Handle("GET /api/v1/admin/debug/pprof/symbol", auth(http.HandlerFunc(h.requireAdmin(pprof.Symbol))))
	mux.Handle("POST /api/v1/admin/debug/pprof/symbol", auth(http.HandlerFunc(h.requireAdmin(pprof.Symbol))))
	mux.Handle("GET /api/v1/admin/debug/pprof/trace", auth(http.HandlerFunc(h.requireAdmin(pprof.Trace))))

	// Memory and goroutine profiles
	mux.Handle("GET /api/v1/admin/debug/pprof/allocs", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("allocs").ServeHTTP))))
	mux.Handle("GET /api/v1/admin/debug/pprof/block", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("block").ServeHTTP))))
	mux.Handle("GET /api/v1/admin/debug/pprof/goroutine", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("goroutine").ServeHTTP))))
	mux.Handle("GET /api/v1/admin/debug/pprof/heap", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("heap").ServeHTTP))))
	mux.Handle("GET /api/v1/admin/debug/pprof/mutex", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("mutex").ServeHTTP))))
	mux.Handle("GET /api/v1/admin/debug/pprof/threadcreate", auth(http.HandlerFunc(h.requireAdmin(pprof.Handler("threadcreate").ServeHTTP))))
}

// getSystemHealth returns system health status.
func (h *AdminHandler) getSystemHealth(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Check database connectivity
	dbStatus := "healthy"
	if err := h.db.Ping(r.Context()); err != nil {
		dbStatus = "unhealthy"
	}

	// Check storage backend
	storageStatus := "healthy"
	if _, err := h.storage.Stats(r.Context()); err != nil {
		storageStatus = "unhealthy"
	}

	status := "healthy"
	if dbStatus != "healthy" || storageStatus != "healthy" {
		status = "degraded"
	}

	// System metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	SuccessResponse(w, map[string]any{
		"status":   status,
		"database": dbStatus,
		"storage":  storageStatus,
		"system": map[string]any{
			"goroutines":       runtime.NumGoroutine(),
			"memory_alloc_mb":  memStats.Alloc / 1024 / 1024,
			"memory_sys_mb":    memStats.Sys / 1024 / 1024,
			"gc_count":         memStats.NumGC,
			"cpu_cores":        runtime.NumCPU(),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// getSystemStats returns system statistics.
func (h *AdminHandler) getSystemStats(w http.ResponseWriter, r *http.Request) {
	// Check admin role
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Get database stats
	dbStats := h.db.Stats()

	// Get user counts
	totalUsers, err := h.db.CountUsersByStatus(r.Context(), "")
	if err != nil {
		log.Printf("admin stats: CountUsersByStatus: %v", err)
	}
	activeUsers, err := h.db.CountUsersByStatus(r.Context(), "active")
	if err != nil {
		log.Printf("admin stats: CountUsersByStatus(active): %v", err)
	}

	// Get storage backend stats
	storageStats, _ := h.storage.Stats(r.Context())
	var totalStorageBytes int64
	if storageStats != nil {
		totalStorageBytes = storageStats.TotalBytes
	}

	// Get file counts and total bytes
	totalFiles, err := h.db.CountFiles(r.Context())
	if err != nil {
		log.Printf("admin stats: CountFiles: %v", err)
	}
	totalBytes, err := h.db.SumFileBytes(r.Context())
	if err != nil {
		log.Printf("admin stats: SumFileBytes: %v", err)
	}

	// Get share counts
	activeShares, err := h.db.CountActiveShares(r.Context())
	if err != nil {
		log.Printf("admin stats: CountActiveShares: %v", err)
	}

	// Get chunk counts
	totalChunks, err := h.db.GetTotalChunkCount(r.Context())
	if err != nil {
		log.Printf("admin stats: GetTotalChunkCount: %v", err)
	}
	totalChunkSize, err := h.db.GetTotalChunkSize(r.Context())
	if err != nil {
		log.Printf("admin stats: GetTotalChunkSize: %v", err)
	}

	stats := map[string]any{
		"database": map[string]any{
			"max_open_connections": dbStats.MaxOpenConnections,
			"open_connections":     dbStats.OpenConnections,
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
		},
		"users": map[string]int64{
			"total":  totalUsers,
			"active": activeUsers,
		},
		"storage": map[string]any{
			"total_files":       totalFiles,
			"total_bytes":       totalBytes,
			"total_chunks":       totalChunks,
			"total_chunk_size":  totalChunkSize,
			"active_shares":    activeShares,
			"storage_backend": totalStorageBytes,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	SuccessResponse(w, stats)
}

// listUsers returns paginated user list.
func (h *AdminHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Parse pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Get users
	users, total, err := h.db.ListUsers(r.Context(), offset, limit, db.UserFilter{})
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	// Sanitize user data for admin view
	var sanitizedUsers []map[string]any
	for _, user := range users {
		sanitizedUsers = append(sanitizedUsers, map[string]any{
			"id":                  user.ID,
			"username":            user.Username,
			"email":               user.Email,
			"status":              user.Status,
			"role":                user.Role,
			"quota_bytes":         user.QuotaBytes,
			"used_bytes":          user.UsedBytes,
			"totp_enabled":        user.TOTPEnabled,
			"last_login_at":       user.LastLoginAt,
			"created_at":          user.CreatedAt,
		})
	}

	SuccessResponse(w, map[string]any{
		"users":  sanitizedUsers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// createUser creates a new user.
func (h *AdminHandler) createUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		ErrorResponse(w, http.StatusBadRequest, "Username, email and password required")
		return
	}

	// Hash password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	user := &db.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		Status:       "active",
	}

	if err := h.db.CreateUser(r.Context(), user); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	CreatedResponse(w, map[string]string{
		"message": "User created successfully",
		"user_id": user.ID,
	})
}

// getUser returns user details.
func (h *AdminHandler) getUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	userID := r.PathValue("userID")
	if userID == "" {
		ErrorResponse(w, http.StatusBadRequest, "User ID required")
		return
	}

	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	SuccessResponse(w, map[string]any{
		"id":                  user.ID,
		"username":            user.Username,
		"email":               user.Email,
		"status":              user.Status,
		"role":                user.Role,
		"quota_bytes":         user.QuotaBytes,
		"used_bytes":          user.UsedBytes,
		"totp_enabled":        user.TOTPEnabled,
		"last_login_at":       user.LastLoginAt,
		"created_at":          user.CreatedAt,
		"updated_at":          user.UpdatedAt,
	})
}

// updateUser updates user details.
func (h *AdminHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	userID := r.PathValue("userID")
	if userID == "" {
		ErrorResponse(w, http.StatusBadRequest, "User ID required")
		return
	}

	var req struct {
		Email        string `json:"email"`
		Role         string `json:"role"`
		Status       string `json:"status"`
		QuotaBytes   int64  `json:"quota_bytes"`
		ResetPassword string `json:"reset_password,omitempty"`
	}

	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build updates
	updates := make(map[string]any)
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.QuotaBytes > 0 {
		updates["quota_bytes"] = req.QuotaBytes
	}

	if len(updates) > 0 {
		if err := h.db.UpdateUser(r.Context(), userID, updates); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to update user")
			return
		}
	}

	// Handle password reset
	if req.ResetPassword != "" {
		hash, err := auth.HashPassword(req.ResetPassword)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		if err := h.db.UpdateUserPassword(r.Context(), userID, hash); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to reset password")
			return
		}
	}

	SuccessResponse(w, map[string]string{
		"message": "User updated successfully",
	})
}

// deleteUser deletes a user.
func (h *AdminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	userID := r.PathValue("userID")
	if userID == "" {
		ErrorResponse(w, http.StatusBadRequest, "User ID required")
		return
	}

	// Prevent self-deletion
	currentUserID := GetUserIDFromRequest(r)
	if userID == currentUserID {
		ErrorResponse(w, http.StatusBadRequest, "Cannot delete yourself")
		return
	}

	if err := h.db.DeleteUser(r.Context(), userID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	SuccessResponse(w, map[string]string{
		"message": "User deleted successfully",
	})
}

// getAuditLogs returns audit log entries.
func (h *AdminHandler) getAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Parse filters
	userID := r.URL.Query().Get("user_id")
	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Get audit entries
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	entries, total, err := h.db.GetAuditEntries(r.Context(), userIDPtr, action, resourceType,
		time.Time{}, time.Now(), limit, offset)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get audit logs")
		return
	}

	SuccessResponse(w, map[string]any{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// runGC triggers garbage collection.
func (h *AdminHandler) runGC(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	if h.workers != nil {
		if err := h.workers.QueueGC(worker.GCSpec{
			Type:      "orphaned-chunks",
			OlderThan: time.Now().Add(-24 * time.Hour),
		}); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to queue GC job")
			return
		}
	}

	SuccessResponse(w, map[string]string{
		"message": "GC triggered",
		"status":  "running",
	})
}

// runCleanup triggers cleanup operations.
func (h *AdminHandler) runCleanup(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return
	}

	if h.workers != nil {
		if err := h.workers.QueueGC(worker.GCSpec{
			Type:      "trash",
			OlderThan: time.Now().Add(-30 * 24 * time.Hour),
		}); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to queue cleanup job")
			return
		}
	}

	SuccessResponse(w, map[string]string{
		"message": "Cleanup triggered",
		"status":  "running",
	})
}

// isAdmin checks if the current user has admin role.
func (h *AdminHandler) isAdmin(r *http.Request) bool {
	return IsAdmin(r)
}
