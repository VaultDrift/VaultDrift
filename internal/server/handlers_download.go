package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/vaultdrift/vaultdrift/internal/chunk"
	"github.com/vaultdrift/vaultdrift/internal/crypto"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// DownloadHandler handles file download and streaming requests.
type DownloadHandler struct {
	vfs       *vfs.VFS
	db        *db.Manager
	storage   storage.Backend
	masterKey []byte
}

// NewDownloadHandler creates a new download handler.
func NewDownloadHandler(vfsService *vfs.VFS, database *db.Manager, store storage.Backend) *DownloadHandler {
	return &DownloadHandler{
		vfs:     vfsService,
		db:      database,
		storage: store,
	}
}

// SetMasterKey sets the master key for decrypting file keys.
// In production with zero-knowledge, client provides the key.
func (h *DownloadHandler) SetMasterKey(key []byte) {
	h.masterKey = key
}

// RegisterRoutes registers the download routes.
func (h *DownloadHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// Download file (full or with Range header)
	mux.Handle("GET /api/v1/files/{id}/download", auth.RequireAuth(http.HandlerFunc(h.downloadFile)))

	// Stream file (alias for download, for media streaming)
	mux.Handle("GET /api/v1/files/{id}/stream", auth.RequireAuth(http.HandlerFunc(h.streamFile)))

	// Get encrypted chunks info for client-side decryption
	mux.Handle("GET /api/v1/files/{id}/chunks", auth.RequireAuth(http.HandlerFunc(h.getChunksInfo)))

	// Download specific chunk (for client-side reassembly)
	mux.Handle("GET /api/v1/files/{id}/chunks/{hash}", auth.RequireAuth(http.HandlerFunc(h.downloadChunk)))
}

// downloadFile handles file download with support for HTTP Range requests.
func (h *DownloadHandler) downloadFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Get file metadata
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify ownership
	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if file.Type != "file" {
		ErrorResponse(w, http.StatusBadRequest, "Not a file")
		return
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File has no content")
		return
	}

	// Get manifest
	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get file manifest")
		return
	}

	// Parse Range header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, file, manifest, rangeHeader)
		return
	}

	// Full file download
	h.handleFullDownload(w, r, file, manifest)
}

// handleFullDownload handles full file download.
func (h *DownloadHandler) handleFullDownload(w http.ResponseWriter, r *http.Request, file *db.File, manifest *db.Manifest) {
	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.Name))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", file.UpdatedAt.Format(http.TimeFormat))
	if file.Checksum != nil {
		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", *file.Checksum))
	}

	// Check for If-None-Match (ETag caching)
	if file.Checksum != nil {
		if match := r.Header.Get("If-None-Match"); match != "" {
			if match == fmt.Sprintf("\"%s\"", *file.Checksum) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// If encrypted and no master key, return chunks info for client-side decryption
	if file.IsEncrypted && !h.canDecrypt(file) {
		w.Header().Set("Content-Type", "application/json")
		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"message":     "Zero-knowledge encryption: client-side decryption required",
			"chunks":      manifest.Chunks,
			"size":        file.SizeBytes,
			"chunk_count": len(manifest.Chunks),
		})
		return
	}

	// Stream content
	w.WriteHeader(http.StatusOK)

	if file.IsEncrypted {
		// Stream decrypted content
		if err := h.streamDecryptedContent(r.Context(), manifest, file, w); err != nil {
			// Error already written, can't change status
			return
		}
	} else {
		// Stream unencrypted content
		if err := h.streamContent(r.Context(), manifest, w); err != nil {
			return
		}
	}
}

// handleRangeRequest handles HTTP Range request for partial content.
func (h *DownloadHandler) handleRangeRequest(w http.ResponseWriter, r *http.Request, file *db.File, manifest *db.Manifest, rangeHeader string) {
	// Parse Range header (format: "bytes=start-end")
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range header")
		return
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeSpec, ",")

	// Only support single range for now
	if len(ranges) > 1 {
		ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Multiple ranges not supported")
		return
	}

	// Parse start and end
	var start, end int64
	rangeParts := strings.Split(ranges[0], "-")
	if len(rangeParts) != 2 {
		ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range format")
		return
	}

	if rangeParts[0] == "" {
		// Suffix range: "bytes=-500" means last 500 bytes
		suffix, err := strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range suffix")
			return
		}
		start = file.SizeBytes - suffix
		if start < 0 {
			start = 0
		}
		end = file.SizeBytes - 1
	} else {
		var err error
		start, err = strconv.ParseInt(rangeParts[0], 10, 64)
		if err != nil {
			ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range start")
			return
		}
		if rangeParts[1] == "" {
			// Open-ended: "bytes=1000-" means from 1000 to end
			end = file.SizeBytes - 1
		} else {
			end, err = strconv.ParseInt(rangeParts[1], 10, 64)
			if err != nil {
				ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid Range end")
				return
			}
		}
	}

	// Validate range
	if start < 0 || start >= file.SizeBytes || start > end {
		ErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Range not satisfiable")
		return
	}
	if end >= file.SizeBytes {
		end = file.SizeBytes - 1
	}

	contentLength := end - start + 1

	// If encrypted and no key, range requests not supported
	if file.IsEncrypted && !h.canDecrypt(file) {
		ErrorResponse(w, http.StatusNotImplemented, "Range requests not supported for zero-knowledge encrypted files")
		return
	}

	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, file.SizeBytes))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.Name))

	w.WriteHeader(http.StatusPartialContent)

	// Stream the requested range
	if file.IsEncrypted {
		if err := h.streamDecryptedRange(r.Context(), manifest, file, start, end, w); err != nil {
			return
		}
	} else {
		if err := h.streamRange(r.Context(), manifest, start, end, w); err != nil {
			return
		}
	}
}

// streamContent streams unencrypted file content.
func (h *DownloadHandler) streamContent(ctx context.Context, manifest *db.Manifest, w io.Writer) error {
	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", hash, err)
		}

		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}
	}
	return nil
}

// streamDecryptedContent streams decrypted file content.
func (h *DownloadHandler) streamDecryptedContent(ctx context.Context, manifest *db.Manifest, file *db.File, w io.Writer) error {
	// Get file encryption key
	fileKey, err := h.unwrapFileKey(file)
	if err != nil {
		return err
	}

	// Create reassembler
	reassembler := chunk.NewEncryptedReassembler(h.storage)

	// Convert db.Manifest to chunk.Manifest
	chunkManifest := h.convertManifest(manifest)

	// Reassemble and decrypt
	return reassembler.ReassembleDecrypt(ctx, chunkManifest, fileKey, w)
}

// streamRange streams a specific byte range from unencrypted file.
func (h *DownloadHandler) streamRange(ctx context.Context, manifest *db.Manifest, start, end int64, w io.Writer) error {
	var currentOffset int64
	targetBytes := end - start + 1
	bytesWritten := int64(0)

	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", hash, err)
		}

		chunkSize := int64(len(data))
		chunkStart := currentOffset
		chunkEnd := chunkStart + chunkSize

		// Check if this chunk intersects with the requested range
		if chunkEnd <= start || chunkStart >= end+1 {
			currentOffset = chunkEnd
			continue
		}

		// Calculate slice within chunk
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
			return fmt.Errorf("failed to write chunk data: %w", err)
		}

		bytesWritten += int64(len(slice))
		if bytesWritten >= targetBytes {
			break
		}

		currentOffset = chunkEnd
	}

	return nil
}

// streamDecryptedRange streams a specific byte range from encrypted file.
func (h *DownloadHandler) streamDecryptedRange(ctx context.Context, manifest *db.Manifest, file *db.File, start, end int64, w io.Writer) error {
	// Get file encryption key
	fileKey, err := h.unwrapFileKey(file)
	if err != nil {
		return err
	}

	var currentOffset int64
	targetBytes := end - start + 1
	bytesWritten := int64(0)

	for _, hash := range manifest.Chunks {
		// Fetch encrypted chunk
		encryptedData, err := h.storage.Get(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", hash, err)
		}

		// Decrypt chunk
		plaintext, err := crypto.Decrypt(encryptedData, fileKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt chunk %s: %w", hash, err)
		}

		chunkSize := int64(len(plaintext))
		chunkStart := currentOffset
		chunkEnd := chunkStart + chunkSize

		// Check if this chunk intersects with the requested range
		if chunkEnd <= start || chunkStart >= end+1 {
			currentOffset = chunkEnd
			continue
		}

		// Calculate slice within chunk
		sliceStart := int64(0)
		sliceEnd := chunkSize

		if chunkStart < start {
			sliceStart = start - chunkStart
		}
		if chunkEnd > end+1 {
			sliceEnd = end - chunkStart + 1
		}

		slice := plaintext[sliceStart:sliceEnd]
		if _, err := w.Write(slice); err != nil {
			return fmt.Errorf("failed to write chunk data: %w", err)
		}

		bytesWritten += int64(len(slice))
		if bytesWritten >= targetBytes {
			break
		}

		currentOffset = chunkEnd
	}

	return nil
}

// streamFile handles streaming with inline disposition for media playback.
func (h *DownloadHandler) streamFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if file.Type != "file" {
		ErrorResponse(w, http.StatusBadRequest, "Not a file")
		return
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File has no content")
		return
	}

	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get file manifest")
		return
	}

	// Set inline disposition for streaming (browser will display/play)
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.Name))
	w.Header().Set("Accept-Ranges", "bytes")

	// Handle Range header for media seeking
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		h.handleRangeRequest(w, r, file, manifest, rangeHeader)
		return
	}

	// Full stream
	w.WriteHeader(http.StatusOK)
	if file.IsEncrypted {
		if err := h.streamDecryptedContent(r.Context(), manifest, file, w); err != nil {
			return
		}
	} else {
		if err := h.streamContent(r.Context(), manifest, w); err != nil {
			return
		}
	}
}

// getChunksInfo returns chunk information for client-side decryption.
func (h *DownloadHandler) getChunksInfo(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if file.Type != "file" {
		ErrorResponse(w, http.StatusBadRequest, "Not a file")
		return
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File has no content")
		return
	}

	manifest, err := h.db.GetManifest(r.Context(), *file.ManifestID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get file manifest")
		return
	}

	SuccessResponse(w, map[string]interface{}{
		"file_id":     file.ID,
		"file_name":   file.Name,
		"size":        file.SizeBytes,
		"mime_type":   file.MimeType,
		"encrypted":   file.IsEncrypted,
		"chunks":      manifest.Chunks,
		"chunk_count": len(manifest.Chunks),
		"checksum":    file.Checksum,
	})
}

// downloadChunk downloads a specific encrypted chunk.
func (h *DownloadHandler) downloadChunk(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	chunkHash := r.PathValue("hash")

	if fileID == "" || chunkHash == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID and chunk hash required")
		return
	}

	// Verify file ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Get chunk data
	data, err := h.storage.Get(r.Context(), chunkHash)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to get chunk")
		return
	}

	// Set headers for immutable content
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Chunk-Hash", chunkHash)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// canDecrypt returns true if the handler can decrypt files.
func (h *DownloadHandler) canDecrypt(file *db.File) bool {
	if !file.IsEncrypted {
		return true
	}
	return h.masterKey != nil && len(file.EncryptedKey) > 0
}

// unwrapFileKey unwraps the file encryption key using the master key.
func (h *DownloadHandler) unwrapFileKey(file *db.File) ([]byte, error) {
	if !file.IsEncrypted {
		return nil, nil
	}
	if h.masterKey == nil {
		return nil, fmt.Errorf("no master key available for decryption")
	}
	if len(file.EncryptedKey) == 0 {
		return nil, fmt.Errorf("file has no encrypted key")
	}
	return chunk.UnwrapFileKey(file.EncryptedKey, h.masterKey)
}

// convertManifest converts db.Manifest to chunk.Manifest.
func (h *DownloadHandler) convertManifest(dbManifest *db.Manifest) *chunk.Manifest {
	chunks := make([]chunk.ChunkInfo, len(dbManifest.Chunks))
	var offset int64
	for i, hash := range dbManifest.Chunks {
		chunks[i] = chunk.ChunkInfo{
			Hash:   hash,
			Offset: offset,
			Size:   0, // Size will be determined during processing
		}
	}

	return &chunk.Manifest{
		ID:        dbManifest.ID,
		FileID:    dbManifest.FileID,
		Version:   dbManifest.Version,
		Size:      dbManifest.SizeBytes,
		Chunks:    chunks,
		Checksum:  dbManifest.Checksum,
		DeviceID:  dbManifest.DeviceID,
		CreatedAt: dbManifest.CreatedAt,
	}
}
