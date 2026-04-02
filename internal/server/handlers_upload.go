package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/chunk"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// UploadHandler handles chunked upload API requests.
type UploadHandler struct {
	vfs           *vfs.VFS
	db            *db.Manager
	storage       storage.Backend
	chunker       *chunk.Chunker
	sessions      map[string]*UploadSession
	sessionsMutex sync.RWMutex
}

// UploadSession represents an active upload session.
type UploadSession struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	ParentID    string             `json:"parent_id"`
	FileName    string             `json:"file_name"`
	Size        int64              `json:"size"`
	MimeType    string             `json:"mime_type"`
	Checksum    string             `json:"checksum,omitempty"`
	Status      string             `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	ExpiresAt   time.Time          `json:"expires_at"`
	Chunks      map[int]*ChunkInfo `json:"chunks,omitempty"`
	TotalChunks int                `json:"total_chunks"`
	ChunkHashes map[int][]string   `json:"chunk_hashes,omitempty"` // CDC chunk hashes per upload chunk
	ChunksMutex sync.RWMutex       `json:"-"`
}

// ChunkInfo represents information about an uploaded chunk.
type ChunkInfo struct {
	Index      int       `json:"index"`
	Size       int       `json:"size"`
	Checksum   string    `json:"checksum,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// createUploadRequest represents a request to create an upload session.
type createUploadRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Checksum string `json:"checksum,omitempty"`
}

// createUploadResponse represents the response for a created upload session.
type createUploadResponse struct {
	SessionID   string `json:"session_id"`
	ChunkSize   int    `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
	ExpiresAt   string `json:"expires_at"`
}

// uploadStatusResponse represents the upload status response.
type uploadStatusResponse struct {
	SessionID      string `json:"session_id"`
	Status         string `json:"status"`
	UploadedBytes  int64  `json:"uploaded_bytes"`
	TotalBytes     int64  `json:"total_bytes"`
	MissingChunks  []int  `json:"missing_chunks,omitempty"`
	UploadedChunks []int  `json:"uploaded_chunks,omitempty"`
}

const (
	// DefaultChunkSize is the default size for each chunk (4MB)
	DefaultChunkSize = 4 * 1024 * 1024
	// SessionTTL is how long upload sessions remain valid
	SessionTTL = 24 * time.Hour
	// MaxUploadSize is the maximum file size (10GB)
	MaxUploadSize = 10 * 1024 * 1024 * 1024
)

// NewUploadHandler creates a new upload handler.
func NewUploadHandler(vfsService *vfs.VFS, database *db.Manager, store storage.Backend) *UploadHandler {
	h := &UploadHandler{
		vfs:           vfsService,
		db:            database,
		storage:       store,
		chunker:       chunk.DefaultChunker(),
		sessions:      make(map[string]*UploadSession),
	}

	// Start cleanup goroutine
	go h.cleanupExpiredSessions()

	return h
}

// RegisterRoutes registers the upload routes.
func (h *UploadHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// Create upload session
	mux.Handle("POST /api/v1/uploads", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.createUpload))))

	// Upload chunk
	mux.Handle("PUT /api/v1/uploads/{id}/chunks/{index}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.uploadChunk))))

	// Complete upload
	mux.Handle("POST /api/v1/uploads/{id}/complete", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.completeUpload))))

	// Get upload status
	mux.Handle("GET /api/v1/uploads/{id}/status", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.getUploadStatus))))

	// Cancel/delete upload session
	mux.Handle("DELETE /api/v1/uploads/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.cancelUpload))))
}

// createUpload creates a new upload session.
func (h *UploadHandler) createUpload(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" {
		ErrorResponse(w, http.StatusBadRequest, "Name is required")
		return
	}

	if req.Size <= 0 {
		ErrorResponse(w, http.StatusBadRequest, "Invalid file size")
		return
	}

	if req.Size > MaxUploadSize {
		ErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("File too large. Maximum size is %d bytes", MaxUploadSize))
		return
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to create upload session")
		return
	}

	// Calculate number of chunks
	totalChunks := int((req.Size + DefaultChunkSize - 1) / DefaultChunkSize)

	// Create session
	session := &UploadSession{
		ID:          sessionID,
		UserID:      userID,
		ParentID:    req.ParentID,
		FileName:    req.Name,
		Size:        req.Size,
		MimeType:    req.MimeType,
		Checksum:    req.Checksum,
		Status:      "pending",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SessionTTL),
		Chunks:      make(map[int]*ChunkInfo),
		TotalChunks: totalChunks,
		ChunkHashes: make(map[int][]string),
	}

	h.sessionsMutex.Lock()
	h.sessions[sessionID] = session
	h.sessionsMutex.Unlock()

	resp := createUploadResponse{
		SessionID:   sessionID,
		ChunkSize:   DefaultChunkSize,
		TotalChunks: totalChunks,
		ExpiresAt:   session.ExpiresAt.Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusCreated)
	SuccessResponse(w, resp)
}

// uploadChunk handles chunk upload.
func (h *UploadHandler) uploadChunk(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Session ID required")
		return
	}

	chunkIndexStr := r.PathValue("index")
	if chunkIndexStr == "" {
		ErrorResponse(w, http.StatusBadRequest, "Chunk index required")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		ErrorResponse(w, http.StatusBadRequest, "Invalid chunk index")
		return
	}

	// Get session
	h.sessionsMutex.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMutex.RUnlock()

	if !exists {
		ErrorResponse(w, http.StatusNotFound, "Upload session not found")
		return
	}

	// Verify ownership
	if session.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		ErrorResponse(w, http.StatusGone, "Upload session expired")
		return
	}

	// Validate chunk index
	if chunkIndex >= session.TotalChunks {
		ErrorResponse(w, http.StatusBadRequest, "Invalid chunk index")
		return
	}

	// Check if chunk already uploaded
	session.ChunksMutex.RLock()
	_, alreadyUploaded := session.Chunks[chunkIndex]
	session.ChunksMutex.RUnlock()

	if alreadyUploaded {
		SuccessResponse(w, map[string]string{"status": "already_uploaded"})
		return
	}

	// Read chunk data
	defer r.Body.Close()

	// Limit chunk size
	maxChunkSize := DefaultChunkSize + 1024 // Allow some buffer
	if r.ContentLength > int64(maxChunkSize) {
		ErrorResponse(w, http.StatusBadRequest, "Chunk too large")
		return
	}

	// Read chunk data with size limit
	chunkData := make([]byte, r.ContentLength)
	n, err := r.Body.Read(chunkData)
	if err != nil && err.Error() != "EOF" {
		ErrorResponse(w, http.StatusBadRequest, "Failed to read chunk data")
		return
	}
	chunkData = chunkData[:n]

	// Process chunk with CDC (Content-Defined Chunking)
	cdcChunks, cdcData, err := h.chunker.ChunkWithData(bytes.NewReader(chunkData))
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to process chunk with CDC")
		return
	}

	// Process each CDC chunk: deduplicate and store
	chunkHashes := make([]string, 0, len(cdcChunks))
	for i, cdcChunk := range cdcChunks {
		chunkHashes = append(chunkHashes, cdcChunk.Hash)

		// Check if chunk already exists (deduplication)
		exists, err := h.db.ChunkExists(r.Context(), cdcChunk.Hash)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to check chunk existence")
			return
		}

		if exists {
			// Chunk exists, increment reference count
			if err := h.db.IncrementRefCount(r.Context(), cdcChunk.Hash); err != nil {
				ErrorResponse(w, http.StatusInternalServerError, "Failed to update chunk ref count")
				return
			}
			continue
		}

		// New chunk - store in storage
		if err := h.storage.Put(r.Context(), cdcChunk.Hash, cdcData[i]); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to store chunk")
			return
		}

		// Create chunk record in database
		dbChunk := &db.Chunk{
			Hash:           cdcChunk.Hash,
			SizeBytes:      int64(cdcChunk.Size),
			StorageBackend: "local", // TODO: Get from config
			StoragePath:    cdcChunk.Hash[:2] + "/" + cdcChunk.Hash + ".chunk",
			RefCount:       1,
			IsEncrypted:    false,
			CreatedAt:      time.Now(),
		}
		if err := h.db.CreateChunk(r.Context(), dbChunk); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to create chunk record")
			return
		}
	}

	// Record chunk upload
	session.ChunksMutex.Lock()
	session.Chunks[chunkIndex] = &ChunkInfo{
		Index:      chunkIndex,
		Size:       n,
		UploadedAt: time.Now(),
	}
	session.ChunkHashes[chunkIndex] = chunkHashes

	// Update status if all chunks uploaded
	if len(session.Chunks) == session.TotalChunks {
		session.Status = "completed"
	} else {
		session.Status = "in_progress"
	}
	session.ChunksMutex.Unlock()

	SuccessResponse(w, map[string]any{
		"status":           "uploaded",
		"chunk_index":      chunkIndex,
		"bytes_received":   n,
		"cdc_chunks":       len(cdcChunks),
		"chunk_hashes":     chunkHashes,
	})
}

// completeUpload completes the upload and assembles the file.
func (h *UploadHandler) completeUpload(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Session ID required")
		return
	}

	// Get session
	h.sessionsMutex.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMutex.RUnlock()

	if !exists {
		ErrorResponse(w, http.StatusNotFound, "Upload session not found")
		return
	}

	// Verify ownership
	if session.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check all chunks uploaded
	session.ChunksMutex.RLock()
	uploadedChunks := len(session.Chunks)
	session.ChunksMutex.RUnlock()

	if uploadedChunks < session.TotalChunks {
		missing := h.getMissingChunks(session)
		ErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Missing chunks: %v", missing))
		return
	}

	// Create file in VFS
	file, err := h.vfs.CreateFile(r.Context(), userID, session.ParentID, session.FileName, session.MimeType, session.Size)
	if err != nil {
		if err == vfs.ErrAlreadyExists {
			ErrorResponse(w, http.StatusConflict, "File already exists")
			return
		}
		if err == vfs.ErrInvalidName {
			ErrorResponse(w, http.StatusBadRequest, "Invalid file name")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Collect all chunk hashes from all upload chunks
	session.ChunksMutex.RLock()
	allChunkHashes := make([]string, 0)
	for i := 0; i < session.TotalChunks; i++ {
		hashes, exists := session.ChunkHashes[i]
		if exists {
			allChunkHashes = append(allChunkHashes, hashes...)
		}
	}
	session.ChunksMutex.RUnlock()

	// Create manifest for the file
	manifestID := "manifest_" + generateRandomString(16)
	manifest := &db.Manifest{
		ID:         manifestID,
		FileID:     file.ID,
		Version:    1,
		SizeBytes:  session.Size,
		Chunks:     allChunkHashes,
		ChunkCount: len(allChunkHashes),
		Checksum:   session.Checksum,
		DeviceID:   "server",
		CreatedAt:  time.Now(),
	}

	if err := h.db.CreateManifest(r.Context(), manifest); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to create manifest")
		return
	}

	// Update file with manifest ID
	if err := h.db.UpdateFile(r.Context(), file.ID, map[string]any{
		"manifest_id": manifestID,
	}); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to update file with manifest")
		return
	}

	// Clean up session
	h.sessionsMutex.Lock()
	delete(h.sessions, sessionID)
	h.sessionsMutex.Unlock()

	SuccessResponse(w, map[string]any{
		"status": "completed",
		"file":   file,
	})
}

// getUploadStatus returns the upload status.
func (h *UploadHandler) getUploadStatus(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Session ID required")
		return
	}

	// Get session
	h.sessionsMutex.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMutex.RUnlock()

	if !exists {
		ErrorResponse(w, http.StatusNotFound, "Upload session not found")
		return
	}

	// Verify ownership
	if session.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		ErrorResponse(w, http.StatusGone, "Upload session expired")
		return
	}

	// Calculate uploaded bytes
	session.ChunksMutex.RLock()
	var uploadedBytes int64
	uploadedChunks := make([]int, 0, len(session.Chunks))
	for index, chunk := range session.Chunks {
		uploadedBytes += int64(chunk.Size)
		uploadedChunks = append(uploadedChunks, index)
	}
	missingChunks := h.getMissingChunksLocked(session)
	session.ChunksMutex.RUnlock()

	resp := uploadStatusResponse{
		SessionID:      sessionID,
		Status:         session.Status,
		UploadedBytes:  uploadedBytes,
		TotalBytes:     session.Size,
		MissingChunks:  missingChunks,
		UploadedChunks: uploadedChunks,
	}

	SuccessResponse(w, resp)
}

// cancelUpload cancels and deletes an upload session.
func (h *UploadHandler) cancelUpload(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Session ID required")
		return
	}

	// Get session
	h.sessionsMutex.RLock()
	session, exists := h.sessions[sessionID]
	h.sessionsMutex.RUnlock()

	if !exists {
		ErrorResponse(w, http.StatusNotFound, "Upload session not found")
		return
	}

	// Verify ownership
	if session.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if session expired - still allow cancellation to clean up
	// (no expiration check for cancel)

	// Clean up session and any uploaded chunks
	h.sessionsMutex.Lock()
	delete(h.sessions, sessionID)
	h.sessionsMutex.Unlock()

	// TODO: Clean up uploaded chunks from storage

	SuccessResponse(w, map[string]string{"status": "cancelled"})
}

// getMissingChunks returns a list of missing chunk indices.
func (h *UploadHandler) getMissingChunks(session *UploadSession) []int {
	session.ChunksMutex.RLock()
	defer session.ChunksMutex.RUnlock()
	return h.getMissingChunksLocked(session)
}

// getMissingChunksLocked returns missing chunks (must hold read lock).
func (h *UploadHandler) getMissingChunksLocked(session *UploadSession) []int {
	missing := make([]int, 0)
	for i := 0; i < session.TotalChunks; i++ {
		if _, exists := session.Chunks[i]; !exists {
			missing = append(missing, i)
		}
	}
	return missing
}

// cleanupExpiredSessions periodically removes expired upload sessions.
func (h *UploadHandler) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		h.sessionsMutex.Lock()
		for id, session := range h.sessions {
			if now.After(session.ExpiresAt) {
				delete(h.sessions, id)
			}
		}
		h.sessionsMutex.Unlock()
	}
}

// generateSessionID generates a unique session ID.
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "upload_" + hex.EncodeToString(bytes), nil
}
