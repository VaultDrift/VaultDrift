package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
// The wrap function is applied to each route for per-route middleware (e.g. rate limiting).
// Pass nil if no per-route middleware is needed.
func (h *PublicShareHandler) RegisterRoutes(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	register := func(pattern string, handler http.HandlerFunc) {
		var h http.Handler = handler
		if wrap != nil {
			h = wrap(h)
		}
		mux.Handle(pattern, h)
	}

	// Access share info (metadata)
	register("GET /s/{token}", h.getShareInfo)

	// Download shared file
	register("GET /s/{token}/download", h.downloadSharedFile)

	// Stream shared file (for media preview)
	register("GET /s/{token}/stream", h.streamSharedFile)
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

// sharePasswordRequest is used to parse password from JSON request body.
type sharePasswordRequest struct {
	Password string `json:"password"`
}

// extractSharePassword extracts the share password from the request.
// It checks, in order: JSON request body, Authorization Bearer header, URL query parameter.
func extractSharePassword(r *http.Request) string {
	// 1. Try JSON body (only if Content-Type suggests JSON and body is present)
	if r.Body != nil && r.ContentLength > 0 {
		ct := r.Header.Get("Content-Type")
		if ct == "" || strings.HasPrefix(ct, "application/json") {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil && len(bodyBytes) > 0 {
				var req sharePasswordRequest
				if json.Unmarshal(bodyBytes, &req) == nil && req.Password != "" {
					return req.Password
				}
			}
		}
	}

	// 2. Try Authorization: Bearer <password> header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != "" {
			return token
		}
	}

	// 3. Fall back to query parameter (backward compatibility)
	if pw := r.URL.Query().Get("password"); pw != "" {
		return pw
	}

	return ""
}

// authenticateSharePassword checks whether the request supplies the correct password
// for a password-protected share. It writes the appropriate error response and returns
// false if authentication fails or is required but missing.
func authenticateSharePassword(w http.ResponseWriter, r *http.Request, share *db.Share) bool {
	if share.PasswordHash == nil {
		return true
	}
	password := extractSharePassword(r)
	if password == "" {
		w.Header().Set("WWW-Authenticate", "Bearer realm=\"share\"")
		http.Error(w, "Password required", http.StatusUnauthorized)
		return false
	}
	valid, err := auth.VerifyPassword(password, *share.PasswordHash)
	if err != nil || !valid {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return false
	}
	return true
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
		http.Error(w, "Share not found or unavailable", http.StatusNotFound)
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
		http.Error(w, "Share not found or unavailable", http.StatusNotFound)
		return
	}

	// Check password if required
	if !authenticateSharePassword(w, r, share) {
		return
	}

	// Check preview-only restriction
	if share.PreviewOnly {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Preview only - download not allowed"}`, http.StatusForbidden)
		return
	}

		// Atomically increment download count and check limit
		allowed, err := h.db.IncrementShareDownloadCount(r.Context(), share.ID, share.MaxDownloads)
		if err != nil {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "Download limit reached", http.StatusForbidden)
			return
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
		http.Error(w, "Share not found or unavailable", http.StatusNotFound)
		return
	}

	// Check password if required
	if !authenticateSharePassword(w, r, share) {
		return
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
		if _, err := w.Write(data); err != nil {
			return
		}
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
		if _, err := w.Write(slice); err != nil {
			return
		}

		bytesWritten += int64(len(slice))
		if bytesWritten >= targetBytes {
			break
		}

		currentOffset = chunkEnd
	}
}
