package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
)

// PublicShareHandler handles public share link access (no authentication required).
type PublicShareHandler struct {
	db      *db.Manager
	storage storage.Backend
}

// NewPublicShareHandler creates a new public share handler.
func NewPublicShareHandler(database *db.Manager, store storage.Backend) *PublicShareHandler {
	return &PublicShareHandler{
		db:      database,
		storage: store,
	}
}

// RegisterRoutes registers public share routes (no auth required).
func (h *PublicShareHandler) RegisterRoutes(mux *http.ServeMux) {
	// Access share info (metadata)
	mux.HandleFunc("GET /s/{token}", h.getShareInfo)

	// Download shared file
	mux.HandleFunc("GET /s/{token}/download", h.downloadSharedFile)

	// Stream shared file (for media preview)
	mux.HandleFunc("GET /s/{token}/stream", h.streamSharedFile)
}

// shareInfoResponse represents public share metadata.
type shareInfoResponse struct {
	FileName    string     `json:"file_name"`
	FileType    string     `json:"file_type"`
	SizeBytes   int64      `json:"size_bytes"`
	MimeType    string     `json:"mime_type"`
	PreviewOnly bool       `json:"preview_only"`
	AllowUpload bool       `json:"allow_upload"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	HasPassword bool       `json:"has_password"`
}

// getShareInfo returns public share metadata (no password required for basic info).
func (h *PublicShareHandler) getShareInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	share, file, err := h.validateShare(r, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Increment view count
	_ = h.db.IncrementShareViewCount(r.Context(), share.ID)

	resp := shareInfoResponse{
		FileName:    file.Name,
		FileType:    file.Type,
		SizeBytes:   file.SizeBytes,
		MimeType:    file.MimeType,
		PreviewOnly: share.PreviewOnly,
		AllowUpload: share.AllowUpload,
		ExpiresAt:   share.ExpiresAt,
		HasPassword: share.PasswordHash != nil,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// downloadSharedFile handles file download via share token.
func (h *PublicShareHandler) downloadSharedFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	share, file, err := h.validateShare(r, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check password if required
	if share.PasswordHash != nil {
		password := r.URL.Query().Get("password")
		if password == "" {
			w.Header().Set("WWW-Authenticate", "Bearer realm=\"share\"")
			http.Error(w, "Password required", http.StatusUnauthorized)
			return
		}
		// Verify password hash
		valid, err := auth.VerifyPassword(password, *share.PasswordHash)
		if err != nil || !valid {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Check preview-only restriction
	if share.PreviewOnly {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Preview only - download not allowed"}`, http.StatusForbidden)
		return
	}

	// Check download limit
	if share.MaxDownloads != nil && share.DownloadCount >= *share.MaxDownloads {
		http.Error(w, "Download limit reached", http.StatusForbidden)
		return
	}

	// Increment download count
	if err := h.db.IncrementShareDownloadCount(r.Context(), share.ID); err != nil {
		// Log but don't fail
	}

	// Get manifest
	if file.ManifestID == nil || *file.ManifestID == "" {
		http.Error(w, "File has no content", http.StatusBadRequest)
		return
	}

	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		http.Error(w, "Failed to get file manifest", http.StatusInternalServerError)
		return
	}

	// Parse Range header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, file, manifest, rangeHeader)
		return
	}

	// Full download
	h.handleFullDownload(w, r, file, manifest)
}

// streamSharedFile handles streaming for shared files (for preview).
func (h *PublicShareHandler) streamSharedFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	share, file, err := h.validateShare(r, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check password if required
	if share.PasswordHash != nil {
		password := r.URL.Query().Get("password")
		if password == "" {
			w.Header().Set("WWW-Authenticate", "Bearer realm=\"share\"")
			http.Error(w, "Password required", http.StatusUnauthorized)
			return
		}
		// Verify password hash
		valid, err := auth.VerifyPassword(password, *share.PasswordHash)
		if err != nil || !valid {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Get manifest
	if file.ManifestID == nil || *file.ManifestID == "" {
		http.Error(w, "File has no content", http.StatusBadRequest)
		return
	}

	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		http.Error(w, "Failed to get file manifest", http.StatusInternalServerError)
		return
	}

	// Set inline disposition for preview
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", sanitizeFilename(file.Name)))
	w.Header().Set("Accept-Ranges", "bytes")

	// Handle Range for seeking
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, file, manifest, rangeHeader)
		return
	}

	// Stream full file
	w.WriteHeader(http.StatusOK)
	h.streamContent(r.Context(), manifest, w)
}

// validateShare validates a share token and returns the share and file.
func (h *PublicShareHandler) validateShare(r *http.Request, token string) (*db.Share, *db.File, error) {
	share, err := h.db.GetShareByToken(r.Context(), token)
	if err != nil {
		return nil, nil, fmt.Errorf("share not found")
	}

	// Check expiration
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, nil, fmt.Errorf("share expired")
	}

	// Check if active
	if !share.IsActive {
		return nil, nil, fmt.Errorf("share revoked")
	}

	// Get file
	file, err := h.db.GetFileByID(r.Context(), share.FileID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found")
	}

	return share, file, nil
}

// handleFullDownload handles full file download for public shares.
func (h *PublicShareHandler) handleFullDownload(w http.ResponseWriter, r *http.Request, file *db.File, manifest *db.Manifest) {
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", sanitizeFilename(file.Name)))
	w.Header().Set("Accept-Ranges", "bytes")

	w.WriteHeader(http.StatusOK)
	h.streamContent(r.Context(), manifest, w)
}

// handleRangeRequest handles HTTP Range for public shares.
func (h *PublicShareHandler) handleRangeRequest(w http.ResponseWriter, r *http.Request, file *db.File, manifest *db.Manifest, rangeHeader string) {
	// Simple range parsing
	var start, end int64
	_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

	if end == 0 || end >= file.SizeBytes {
		end = file.SizeBytes - 1
	}

	contentLength := end - start + 1

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.SizeBytes))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)

	h.streamRange(r.Context(), manifest, start, end, w)
}

// streamContent streams unencrypted chunks.
func (h *PublicShareHandler) streamContent(ctx context.Context, manifest *db.Manifest, w http.ResponseWriter) {
	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(ctx, hash)
		if err != nil {
			return
		}
		_, _ = w.Write(data)
	}
}

// streamRange streams a byte range from chunks.
func (h *PublicShareHandler) streamRange(ctx context.Context, manifest *db.Manifest, start, end int64, w http.ResponseWriter) {
	var currentOffset int64
	targetBytes := end - start + 1
	bytesWritten := int64(0)

	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(ctx, hash)
		if err != nil {
			return
		}

		chunkSize := int64(len(data))
		chunkStart := currentOffset
		chunkEnd := chunkStart + chunkSize

		if chunkEnd <= start || chunkStart >= end+1 {
			currentOffset = chunkEnd
			continue
		}

		sliceStart := int64(0)
		sliceEnd := chunkSize

		if chunkStart < start {
			sliceStart = start - chunkStart
		}
		if chunkEnd > end+1 {
			sliceEnd = end - chunkStart + 1
		}

		slice := data[sliceStart:sliceEnd]
		_, _ = w.Write(slice)

		bytesWritten += int64(len(slice))
		if bytesWritten >= targetBytes {
			break
		}

		currentOffset = chunkEnd
	}
}
