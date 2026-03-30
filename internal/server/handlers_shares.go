package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// ShareHandler handles share link API requests.
type ShareHandler struct {
	vfs *vfs.VFS
	db  *db.Manager
}

// NewShareHandler creates a new share handler.
func NewShareHandler(vfsService *vfs.VFS, database *db.Manager) *ShareHandler {
	return &ShareHandler{
		vfs: vfsService,
		db:  database,
	}
}

// RegisterRoutes registers the share routes.
func (h *ShareHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// Create share link
	mux.Handle("POST /api/v1/files/{id}/shares", auth.RequireAuth(http.HandlerFunc(h.createShare)))

	// List shares for a file
	mux.Handle("GET /api/v1/files/{id}/shares", auth.RequireAuth(http.HandlerFunc(h.listShares)))

	// List all shares created by user
	mux.Handle("GET /api/v1/shares", auth.RequireAuth(http.HandlerFunc(h.listMyShares)))

	// List shares received by user
	mux.Handle("GET /api/v1/shares/received", auth.RequireAuth(http.HandlerFunc(h.listReceivedShares)))

	// Get share details
	mux.Handle("GET /api/v1/shares/{id}", auth.RequireAuth(http.HandlerFunc(h.getShare)))

	// Update share
	mux.Handle("PUT /api/v1/shares/{id}", auth.RequireAuth(http.HandlerFunc(h.updateShare)))

	// Revoke/delete share
	mux.Handle("DELETE /api/v1/shares/{id}", auth.RequireAuth(http.HandlerFunc(h.revokeShare)))
}

// createShareRequest represents a request to create a share.
type createShareRequest struct {
	ShareType    string  `json:"share_type"`              // "link" or "user"
	SharedWith   *string `json:"shared_with,omitempty"`   // User ID for user shares
	Password     *string `json:"password,omitempty"`      // Optional password for link shares
	ExpiresDays  *int    `json:"expires_days,omitempty"`  // Days until expiration
	MaxDownloads *int    `json:"max_downloads,omitempty"` // Download limit
	AllowUpload  bool    `json:"allow_upload"`            // Allow uploads to folder
	PreviewOnly  bool    `json:"preview_only"`            // Preview without download
	Permission   string  `json:"permission"`              // "read" or "write"
}

// createShare creates a new share link.
func (h *ShareHandler) createShare(w http.ResponseWriter, r *http.Request) {
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

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate share type
	if req.ShareType != "link" && req.ShareType != "user" {
		ErrorResponse(w, http.StatusBadRequest, "Share type must be 'link' or 'user'")
		return
	}

	// For user shares, require shared_with
	if req.ShareType == "user" && req.SharedWith == nil {
		ErrorResponse(w, http.StatusBadRequest, "shared_with is required for user shares")
		return
	}

	// Default permission
	if req.Permission == "" {
		req.Permission = "read"
	}

	// Generate share token for link shares
	var token *string
	if req.ShareType == "link" {
		t, err := generateShareToken()
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError, "Failed to generate share token")
			return
		}
		token = &t
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresDays != nil && *req.ExpiresDays > 0 {
		t := time.Now().UTC().Add(time.Duration(*req.ExpiresDays) * 24 * time.Hour)
		expiresAt = &t
	}

	// Create share record
	share := &db.Share{
		ID:           generateShareID(),
		FileID:       fileID,
		CreatedBy:    userID,
		ShareType:    req.ShareType,
		Token:        token,
		PasswordHash: req.Password, // TODO: Hash the password
		ExpiresAt:    expiresAt,
		MaxDownloads: req.MaxDownloads,
		AllowUpload:  req.AllowUpload,
		PreviewOnly:  req.PreviewOnly,
		SharedWith:   req.SharedWith,
		Permission:   req.Permission,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.db.CreateShare(r.Context(), share); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to create share")
		return
	}

	// Construct share URL for link shares
	var shareURL *string
	if token != nil {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		url := scheme + "://" + r.Host + "/s/" + *token
		shareURL = &url
	}

	w.WriteHeader(http.StatusCreated)
	SuccessResponse(w, map[string]any{
		"share":     share,
		"share_url": shareURL,
	})
}

// listShares lists all shares for a file.
func (h *ShareHandler) listShares(w http.ResponseWriter, r *http.Request) {
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

	shares, err := h.db.GetSharesByFile(r.Context(), fileID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list shares")
		return
	}

	SuccessResponse(w, map[string]any{
		"shares": shares,
	})
}

// listMyShares lists all shares created by the current user.
func (h *ShareHandler) listMyShares(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shares, err := h.db.GetSharesByUser(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list shares")
		return
	}

	SuccessResponse(w, map[string]any{
		"shares": shares,
	})
}

// listReceivedShares lists shares shared with the current user.
func (h *ShareHandler) listReceivedShares(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shares, err := h.db.GetReceivedShares(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to list shares")
		return
	}

	SuccessResponse(w, map[string]any{
		"shares": shares,
	})
}

// getShare returns share details.
func (h *ShareHandler) getShare(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shareID := r.PathValue("id")
	if shareID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Share ID required")
		return
	}

	share, err := h.db.GetShareByID(r.Context(), shareID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Share not found")
		return
	}

	// Verify ownership or recipient
	if share.CreatedBy != userID && (share.SharedWith == nil || *share.SharedWith != userID) {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	SuccessResponse(w, share)
}

// updateShareRequest represents a share update request.
type updateShareRequest struct {
	MaxDownloads *int    `json:"max_downloads,omitempty"`
	AllowUpload  *bool   `json:"allow_upload,omitempty"`
	PreviewOnly  *bool   `json:"preview_only,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

// updateShare updates share settings.
func (h *ShareHandler) updateShare(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shareID := r.PathValue("id")
	if shareID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Share ID required")
		return
	}

	share, err := h.db.GetShareByID(r.Context(), shareID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Share not found")
		return
	}

	// Only creator can update
	if share.CreatedBy != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	var req updateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updates := make(map[string]any)
	if req.MaxDownloads != nil {
		updates["max_downloads"] = *req.MaxDownloads
	}
	if req.AllowUpload != nil {
		updates["allow_upload"] = *req.AllowUpload
	}
	if req.PreviewOnly != nil {
		updates["preview_only"] = *req.PreviewOnly
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.UpdateShare(r.Context(), shareID, updates); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to update share")
		return
	}

	SuccessResponse(w, map[string]string{"status": "updated"})
}

// revokeShare revokes/deletes a share.
func (h *ShareHandler) revokeShare(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	shareID := r.PathValue("id")
	if shareID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Share ID required")
		return
	}

	share, err := h.db.GetShareByID(r.Context(), shareID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "Share not found")
		return
	}

	// Only creator can revoke
	if share.CreatedBy != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.db.RevokeShare(r.Context(), shareID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to revoke share")
		return
	}

	SuccessResponse(w, map[string]string{"status": "revoked"})
}

// Helper functions

func generateShareID() string {
	return "share_" + generateRandomString(16)
}

func generateShareToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
