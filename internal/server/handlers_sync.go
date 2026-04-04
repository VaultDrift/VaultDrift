package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	syncpkg "github.com/vaultdrift/vaultdrift/internal/sync"
)

// SyncHandler handles sync protocol operations.
type SyncHandler struct {
	engine *syncpkg.Engine
	db     *db.Manager
	store  storage.Backend
}

// NewSyncHandler creates a new sync handler.
func NewSyncHandler(engine *syncpkg.Engine, database *db.Manager, store storage.Backend) *SyncHandler {
	return &SyncHandler{
		engine: engine,
		db:     database,
		store:  store,
	}
}

// RegisterRoutes registers sync API routes.
func (h *SyncHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// Device management
	mux.Handle("POST /api/v1/sync/devices", auth(http.HandlerFunc(h.registerDevice)))
	mux.Handle("GET /api/v1/sync/devices", auth(http.HandlerFunc(h.listDevices)))
	mux.Handle("DELETE /api/v1/sync/devices/{deviceID}", auth(http.HandlerFunc(h.removeDevice)))
	mux.Handle("GET /api/v1/sync/sessions", auth(http.HandlerFunc(h.listSessions)))

	// Sync protocol
	mux.Handle("POST /api/v1/sync/negotiate", auth(http.HandlerFunc(h.negotiate)))
	mux.Handle("POST /api/v1/sync/push", auth(http.HandlerFunc(h.push)))
	mux.Handle("GET /api/v1/sync/pull/{hash}", auth(http.HandlerFunc(h.pullChunk)))
	mux.Handle("POST /api/v1/sync/commit", auth(http.HandlerFunc(h.commit)))
	mux.Handle("GET /api/v1/sync/status", auth(http.HandlerFunc(h.getStatus)))
}

// registerDevice registers or updates a sync device.
func (h *SyncHandler) registerDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		DeviceID   string `json:"device_id"`
		Name       string `json:"name"`
		DeviceType string `json:"device_type"`
		OS         string `json:"os"`
		SyncFolder string `json:"sync_folder"`
	}

	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeviceID == "" || req.Name == "" {
		ErrorResponse(w, http.StatusBadRequest, "device_id and name are required")
		return
	}

	device := &db.Device{
		ID:         req.DeviceID,
		UserID:     userID,
		Name:       req.Name,
		DeviceType: req.DeviceType,
		OS:         req.OS,
		SyncFolder: req.SyncFolder,
	}

	if err := h.engine.RegisterDevice(r.Context(), device); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to register device")
		return
	}

	SuccessResponse(w, map[string]string{
		"message":   "Device registered",
		"device_id": device.ID,
	})
}

// listDevices returns all devices for the current user.
func (h *SyncHandler) listDevices(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	devices, err := h.db.GetDevicesByUser(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list devices")
		return
	}

	SuccessResponse(w, map[string]any{
		"devices": devices,
	})
}

// removeDevice removes a sync device.
func (h *SyncHandler) removeDevice(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	deviceID := r.PathValue("deviceID")
	if deviceID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Device ID required")
		return
	}

	// Verify device belongs to user
	device, err := h.db.GetDeviceByID(r.Context(), deviceID)
	if err != nil || device.UserID != userID {
		ErrorResponse(w, http.StatusNotFound, "Device not found")
		return
	}

	if err := h.db.DeleteDevice(r.Context(), deviceID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to remove device")
		return
	}

	SuccessResponse(w, map[string]string{
		"message": "Device removed",
	})
}

// negotiate handles sync negotiation.
func (h *SyncHandler) negotiate(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req syncpkg.NegotiateRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeviceID == "" {
		ErrorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Verify device ownership
	if err := h.verifyDeviceOwnership(r.Context(), userID, req.DeviceID); err != nil {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	resp, err := h.engine.Negotiate(r.Context(), userID, req)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Negotiation failed")
		return
	}

	JSONResponse(w, http.StatusOK, resp)
}

// push receives chunk data from a client.
func (h *SyncHandler) push(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse multipart form with chunk data
	if err := r.ParseMultipartForm(64 << 20); err != nil { // 64MB max
		ErrorResponse(w, http.StatusBadRequest, "Failed to parse upload")
		return
	}

	hash := r.FormValue("hash")
	if hash == "" {
		ErrorResponse(w, http.StatusBadRequest, "Chunk hash required")
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Chunk data required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to read chunk data")
		return
	}
	// Verify hash integrity
	computed := sha256.Sum256(data)
	computedHash := hex.EncodeToString(computed[:])
	if computedHash != hash {
		ErrorResponse(w, http.StatusBadRequest, "Chunk hash mismatch")
		return
	}

	if err := h.store.Put(r.Context(), hash, data); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to store chunk")
		return
	}

	SuccessResponse(w, map[string]string{
		"message": "Chunk stored",
		"hash":    hash,
	})
}

// pullChunk streams a chunk to the client.
func (h *SyncHandler) pullChunk(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	hash := r.PathValue("hash")
	if hash == "" {
		ErrorResponse(w, http.StatusBadRequest, "Chunk hash required")
		return
	}

	// Verify the user owns a file that references this chunk
	owns, err := h.db.UserOwnsChunk(r.Context(), userID, hash)
	if err != nil {
		InternalErrorResponse(w, err)
		return
	}
	if !owns {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Retrieve chunk from storage backend
	data, err := h.store.Get(r.Context(), hash)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Chunk not found")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Chunk-Hash", hash)
	w.Header().Set("Cache-Control", "max-age=31536000, immutable")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	if _, err := w.Write(data); err != nil {
		log.Printf("pullChunk: failed to write chunk %s to response: %v", hash, err)
	}
}

// commit applies a batch of file changes.
func (h *SyncHandler) commit(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req syncpkg.CommitRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DeviceID == "" {
		ErrorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Verify device ownership
	if err := h.verifyDeviceOwnership(r.Context(), userID, req.DeviceID); err != nil {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	resp, err := h.engine.Commit(r.Context(), userID, req)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Commit failed")
		return
	}

	status := http.StatusOK
	if resp.Status == "partial" {
		status = http.StatusConflict
	}

	JSONResponse(w, status, resp)
}

// getStatus returns the current sync status for the requesting device.
func (h *SyncHandler) getStatus(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		// Return overall sync status
		SuccessResponse(w, map[string]any{
			"user_id": userID,
			"status":  "ready",
		})
		return
	}

	// Verify device ownership
	if err := h.verifyDeviceOwnership(r.Context(), userID, deviceID); err != nil {
		ErrorResponse(w, http.StatusNotFound, "Device not found")
		return
	}

	status, err := h.engine.GetStatus(r.Context(), deviceID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Device not found")
		return
	}

	SuccessResponse(w, status)
}

// verifyDeviceOwnership checks that the given device belongs to the authenticated user.
func (h *SyncHandler) verifyDeviceOwnership(ctx context.Context, userID, deviceID string) error {
	device, err := h.db.GetDeviceByID(ctx, deviceID)
	if err != nil || device.UserID != userID {
		return fmt.Errorf("device not owned by user")
	}
	return nil
}

// listSessions returns sync state history for the current user, grouped by device.
func (h *SyncHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	devices, err := h.db.GetDevicesByUser(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	type deviceSyncInfo struct {
		DeviceID   string           `json:"device_id"`
		DeviceName string           `json:"device_name"`
		States     []*db.SyncState  `json:"states"`
	}

	sessions := make([]deviceSyncInfo, 0, len(devices))
	for _, d := range devices {
		states, err := h.db.GetSyncStatesByDevice(r.Context(), d.ID)
		if err != nil {
			// Log but continue — a single device failure shouldn't break the list
			log.Printf("listSessions: failed to get states for device %s: %v", d.ID, err)
			states = []*db.SyncState{}
		}
		sessions = append(sessions, deviceSyncInfo{
			DeviceID:   d.ID,
			DeviceName: d.Name,
			States:     states,
		})
	}

	SuccessResponse(w, map[string]any{
		"sessions": sessions,
	})
}
