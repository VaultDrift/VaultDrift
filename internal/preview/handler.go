package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// Handler handles document preview requests
type Handler struct {
	converter *DocumentConverter
	vfs       *vfs.VFS
	db        *db.Manager
	storage   storage.Backend
}

// NewHandler creates a new preview handler
func NewHandler(vfsService *vfs.VFS, database *db.Manager, store storage.Backend) *Handler {
	return &Handler{
		converter: NewDocumentConverter(vfsService, database, store),
		vfs:       vfsService,
		db:        database,
		storage:   store,
	}
}

// RegisterRoutes registers preview endpoints
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/preview/{fileID}", auth(http.HandlerFunc(h.handlePreviewInfo)))
	mux.Handle("GET /api/v1/preview/{fileID}/pdf", auth(http.HandlerFunc(h.handlePreviewPDF)))
	mux.Handle("GET /api/v1/preview/{fileID}/html", auth(http.HandlerFunc(h.handlePreviewHTML)))
	mux.Handle("POST /api/v1/preview/{fileID}/generate", auth(http.HandlerFunc(h.handleGeneratePreview)))
}

// PreviewInfo represents preview metadata
type PreviewInfo struct {
	FileID           string   `json:"file_id"`
	CanPreview       bool     `json:"can_preview"`
	PreviewType      string   `json:"preview_type,omitempty"`
	Status           string   `json:"status"`
	SupportedTypes   []string `json:"supported_types"`
	ConverterEnabled bool     `json:"converter_enabled"`
}

func (h *Handler) handlePreviewInfo(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	info := PreviewInfo{
		FileID:           fileID,
		CanPreview:       h.converter.CanPreview(file.MimeType),
		SupportedTypes:   h.converter.SupportedFormats(),
		ConverterEnabled: h.converter.IsEnabled(),
	}

	// Check if preview already exists
	previewPath := filepath.Join(h.converter.cacheDir, fileID+".pdf")
	if _, err := os.Stat(previewPath); err == nil {
		info.Status = "ready"
		info.PreviewType = "pdf"
	} else if h.converter.CanPreview(file.MimeType) {
		info.Status = "pending"
	} else {
		info.Status = "unsupported"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *Handler) handlePreviewPDF(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if it's a PDF already
	if file.MimeType == "application/pdf" {
		// Stream original file
		h.streamFile(w, r, fileID)
		return
	}

	// Get or generate preview
	result, err := h.converter.GetPreview(r.Context(), fileID)
	if err != nil {
		// Try to generate on-the-fly
		result, err = h.converter.GeneratePreview(r.Context(), fileID)
		if err != nil {
			http.Error(w, "Preview not available: "+err.Error(), http.StatusNotFound)
			return
		}
	}

	// Serve PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, result.PreviewPath)
}

func (h *Handler) handlePreviewHTML(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Generate HTML preview
	htmlPath, err := h.converter.GenerateHTML(r.Context(), fileID)
	if err != nil {
		http.Error(w, "HTML preview not available", http.StatusNotFound)
		return
	}

	// Serve HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, htmlPath)
}

func (h *Handler) handleGeneratePreview(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if previewable
	if !h.converter.CanPreview(file.MimeType) {
		http.Error(w, "File type not supported for preview", http.StatusBadRequest)
		return
	}

	// Generate preview (async)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		h.converter.GeneratePreview(ctx, fileID)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "generating",
		"file_id": fileID,
	})
}

// streamFile streams the original file content
func (h *Handler) streamFile(w http.ResponseWriter, r *http.Request, fileID string) {
	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		http.Error(w, "File has no content", http.StatusNotFound)
		return
	}

	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		http.Error(w, "Failed to get manifest", http.StatusInternalServerError)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.SizeBytes))

	// Stream chunks
	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(r.Context(), hash)
		if err != nil {
			return
		}
		w.Write(data)
	}
}

// getUserID extracts user ID from request context
func getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}
