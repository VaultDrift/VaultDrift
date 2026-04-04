package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/thumbnail"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// ThumbnailHandler handles thumbnail generation and serving
type ThumbnailHandler struct {
	generator *thumbnail.Generator
	vfs       *vfs.VFS
	db        *db.Manager
	storage   storage.Backend
}

// NewThumbnailHandler creates a new thumbnail handler
func NewThumbnailHandler(vfsService *vfs.VFS, database *db.Manager, store storage.Backend, cacheDir string) *ThumbnailHandler {
	gen := thumbnail.NewGenerator(store, cacheDir)
	// Initialize cache directory (ignore error if already exists)
	_ = gen.Init()

	return &ThumbnailHandler{
		generator: gen,
		vfs:       vfsService,
		db:        database,
		storage:   store,
	}
}

// RegisterRoutes registers thumbnail endpoints
func (h *ThumbnailHandler) RegisterRoutes(mux *http.ServeMux, middleware *AuthMiddleware) {
	// Get thumbnail
	mux.Handle("GET /api/v1/thumbnails/{fileID}", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.handleThumbnail))))

	// Generate thumbnail
	mux.Handle("POST /api/v1/thumbnails/{fileID}/generate", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.handleGenerate))))

	// Delete thumbnail
	mux.Handle("DELETE /api/v1/thumbnails/{fileID}", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.handleDelete))))
}

// handleThumbnail serves a thumbnail image
func (h *ThumbnailHandler) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "File not found")
		return
	}

	// Verify ownership
	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get size parameter (default to medium)
	size := r.URL.Query().Get("size")
	if size == "" {
		size = thumbnail.SizeMedium.Name
	}

	// Validate size
	validSize := false
	for _, s := range thumbnail.AllSizes {
		if s.Name == size {
			validSize = true
			break
		}
	}
	if !validSize {
		size = thumbnail.SizeMedium.Name
	}

	// Check if thumbnail exists
	thumbPath, exists := h.generator.Get(fileID, size)
	if !exists {
		// Generate on-the-fly for images
		if h.generator.CanGenerate(file.MimeType) {
			if err := h.generateThumbnail(r.Context(), fileID, file.MimeType); err != nil {
				ErrorResponse(w, http.StatusInternalServerError, "Failed to generate thumbnail")
				return
			}
			thumbPath, exists = h.generator.Get(fileID, size)
			if !exists {
				ErrorResponse(w, http.StatusNotFound, "Thumbnail not available")
				return
			}
		} else {
			ErrorResponse(w, http.StatusBadRequest, "File type not supported for thumbnails")
			return
		}
	}

	// Serve thumbnail
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, thumbPath)
}

// handleGenerate triggers thumbnail generation
func (h *ThumbnailHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "File not found")
		return
	}

	// Verify ownership
	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if supported
	if !h.generator.CanGenerate(file.MimeType) {
		ErrorResponse(w, http.StatusBadRequest, "File type not supported for thumbnails")
		return
	}

	// Generate thumbnails
	if err := h.generateThumbnail(r.Context(), fileID, file.MimeType); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to generate thumbnail")
		return
	}

	SuccessResponse(w, map[string]string{
		"status":  "generated",
		"file_id": fileID,
	})
}

// handleDelete deletes thumbnails for a file
func (h *ThumbnailHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "File not found")
		return
	}

	// Verify ownership
	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Delete thumbnails
	if err := h.generator.Delete(fileID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to delete thumbnails")
		return
	}

	SuccessResponse(w, map[string]string{"status": "deleted"})
}

// generateThumbnail generates thumbnails for a file
func (h *ThumbnailHandler) generateThumbnail(ctx context.Context, fileID, mimeType string) error {
	// Get file data from storage
	// For thumbnails, we need the original file data
	// This is simplified - in production you'd stream from storage
	file, err := h.vfs.GetFile(ctx, fileID)
	if err != nil {
		return err
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		return fmt.Errorf("file has no content")
	}

	manifest, err := h.db.GetManifest(ctx, *file.ManifestID)
	if err != nil {
		return err
	}

	// Check file size before loading into memory (limit: 50MB for thumbnail generation)
	const maxThumbnailFileSize = 50 * 1024 * 1024
	if file.SizeBytes > maxThumbnailFileSize {
		return fmt.Errorf("file too large for thumbnail generation (%d bytes)", file.SizeBytes)
	}

	// Reassemble file data from chunks
	var data []byte
	for _, hash := range manifest.Chunks {
		chunk, err := h.storage.Get(ctx, hash)
		if err != nil {
			return err
		}
		data = append(data, chunk...)
	}

	// Generate thumbnails
	_, err = h.generator.Generate(fileID, mimeType, data)
	return err
}
