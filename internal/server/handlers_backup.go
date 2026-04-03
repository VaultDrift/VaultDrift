package server

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// BackupHandler handles database backup and restore operations.
type BackupHandler struct {
	db        *db.Manager
	dataDir   string
}

// NewBackupHandler creates a new backup handler.
func NewBackupHandler(database *db.Manager, dataDir string) *BackupHandler {
	return &BackupHandler{
		db:      database,
		dataDir: dataDir,
	}
}

// RegisterRoutes registers backup routes (admin only).
func (h *BackupHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// Create backup (admin only)
	mux.Handle("POST /api/v1/admin/backup", auth(http.HandlerFunc(h.createBackup)))

	// List backups (admin only)
	mux.Handle("GET /api/v1/admin/backups", auth(http.HandlerFunc(h.listBackups)))

	// Download backup (admin only)
	mux.Handle("GET /api/v1/admin/backups/{filename}", auth(http.HandlerFunc(h.downloadBackup)))

	// Delete backup (admin only)
	mux.Handle("DELETE /api/v1/admin/backups/{filename}", auth(http.HandlerFunc(h.deleteBackup)))

	// Restore from backup (admin only)
	mux.Handle("POST /api/v1/admin/restore", auth(http.HandlerFunc(h.restoreBackup)))
}

// createBackup creates a new database backup.
func (h *BackupHandler) createBackup(w http.ResponseWriter, r *http.Request) {
	// Check admin role (would be checked by middleware in real implementation)
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Create backup directory if not exists
	backupDir := filepath.Join(h.dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to create backup directory")
		return
	}

	// Generate backup filename
	timestamp := time.Now().UTC().Format("20060102_150405")
	filename := fmt.Sprintf("vaultdrift_backup_%s.tar.gz", timestamp)
	backupPath := filepath.Join(backupDir, filename)

	// Create backup archive
	if err := h.createBackupArchive(backupPath); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create backup: %v", err))
		return
	}

	// Get file info for size
	info, err := os.Stat(backupPath)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get backup info")
		return
	}

	SuccessResponse(w, map[string]any{
		"message":   "Backup created successfully",
		"filename":  filename,
		"size":      info.Size(),
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// createBackupArchive creates a tar.gz archive of the database.
func (h *BackupHandler) createBackupArchive(backupPath string) error {
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Get database path
	dbPath := h.db.Path() // Assumes db.Manager has Path() method
	if dbPath == "" {
		return fmt.Errorf("database path not available")
	}

	// Add database file to archive
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("failed to stat database: %w", err)
	}

	dbFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer dbFile.Close()

	header := &tar.Header{
		Name:    "vaultdrift.db",
		Size:    info.Size(),
		Mode:    0600,
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := io.Copy(tw, dbFile); err != nil {
		return fmt.Errorf("failed to write database to archive: %w", err)
	}

	return nil
}

// listBackups lists available backups.
func (h *BackupHandler) listBackups(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	backupDir := filepath.Join(h.dataDir, "backups")

	// Create directory if not exists (return empty list)
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		SuccessResponse(w, map[string]any{
			"backups": []any{},
		})
		return
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to read backup directory")
		return
	}

	var backups []map[string]any
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, map[string]any{
			"filename":   entry.Name(),
			"size":       info.Size(),
			"created_at": info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	SuccessResponse(w, map[string]any{
		"backups": backups,
	})
}

// downloadBackup serves a backup file for download.
func (h *BackupHandler) downloadBackup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	filename := r.PathValue("filename")
	if filename == "" {
		ErrorResponse(w, http.StatusBadRequest, "Filename required")
		return
	}

	// Sanitize filename to prevent path traversal
	if containsPathTraversal(filename) {
		ErrorResponse(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	backupPath := filepath.Join(h.dataDir, "backups", filename)

	// Verify file exists and is within backup directory
	absPath, err := filepath.Abs(backupPath)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Invalid path")
		return
	}

	absBackupDir, _ := filepath.Abs(filepath.Join(h.dataDir, "backups"))
	if !strings.HasPrefix(absPath, absBackupDir) {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	file, err := os.Open(backupPath)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Backup not found")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get file info")
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	io.Copy(w, file)
}

// deleteBackup removes a backup file.
func (h *BackupHandler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	filename := r.PathValue("filename")
	if filename == "" {
		ErrorResponse(w, http.StatusBadRequest, "Filename required")
		return
	}

	// Sanitize filename
	if containsPathTraversal(filename) {
		ErrorResponse(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	backupPath := filepath.Join(h.dataDir, "backups", filename)

	// Verify path is within backup directory
	absPath, _ := filepath.Abs(backupPath)
	absBackupDir, _ := filepath.Abs(filepath.Join(h.dataDir, "backups"))
	if !strings.HasPrefix(absPath, absBackupDir) {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := os.Remove(backupPath); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to delete backup")
		return
	}

	SuccessResponse(w, map[string]string{
		"message": "Backup deleted successfully",
	})
}

// restoreBackup restores database from a backup.
func (h *BackupHandler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Filename == "" {
		ErrorResponse(w, http.StatusBadRequest, "Filename required")
		return
	}

	// Sanitize filename
	if containsPathTraversal(req.Filename) {
		ErrorResponse(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	backupPath := filepath.Join(h.dataDir, "backups", req.Filename)

	// Verify path is within backup directory
	absPath, _ := filepath.Abs(backupPath)
	absBackupDir, _ := filepath.Abs(filepath.Join(h.dataDir, "backups"))
	if !strings.HasPrefix(absPath, absBackupDir) {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		ErrorResponse(w, http.StatusNotFound, "Backup not found")
		return
	}

	// TODO: Implement actual restore (would require stopping writes, restoring, restarting)
	// For now, return warning
	SuccessResponse(w, map[string]string{
		"message": "Restore endpoint - manual restore required. Use backup file from: " + backupPath,
		"warning": "Database restore requires server restart. Please follow manual restore procedure.",
	})
}

// containsPathTraversal checks for path traversal attempts.
func containsPathTraversal(filename string) bool {
	return strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\")
}

// GetUserID extracts user ID from request context.
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}
