package server

import (
	"encoding/json"
	"net/http"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/db"
)

// UserHandler handles user profile API requests.
type UserHandler struct {
	db      *db.Manager
	authSvc *auth.Service
}

// NewUserHandler creates a new user handler.
func NewUserHandler(database *db.Manager, authService *auth.Service) *UserHandler {
	return &UserHandler{
		db:      database,
		authSvc: authService,
	}
}

// RegisterRoutes registers the user routes.
func (h *UserHandler) RegisterRoutes(mux *http.ServeMux, middleware *AuthMiddleware) {
	// Get user profile
	mux.Handle("GET /api/v1/user/profile", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.getProfile))))

	// Update user profile
	mux.Handle("PUT /api/v1/user/profile", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.updateProfile))))

	// Change password
	mux.Handle("PUT /api/v1/user/password", middleware.Authenticate(middleware.RequireAuth(http.HandlerFunc(h.changePassword))))
}

// getProfile returns the current user's profile.
func (h *UserHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Return safe user info (no password hash)
	SuccessResponse(w, map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"quota_bytes":  user.QuotaBytes,
		"used_bytes":   user.UsedBytes,
		"totp_enabled": user.TOTPEnabled,
		"status":       user.Status,
		"created_at":   user.CreatedAt,
		"last_login":   user.LastLoginAt,
	})
}

// updateProfileRequest represents a profile update request.
type updateProfileRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Email       *string `json:"email,omitempty"`
}

// updateProfile updates the current user's profile.
func (h *UserHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build updates map
	updates := make(map[string]any)
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	if len(updates) == 0 {
		ErrorResponse(w, http.StatusBadRequest, "No fields to update")
		return
	}

	// Update user
	if err := h.db.UpdateUser(r.Context(), userID, updates); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	SuccessResponse(w, map[string]string{"status": "updated"})
}

// changePasswordRequest represents a password change request.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// changePassword changes the current user's password.
func (h *UserHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		ErrorResponse(w, http.StatusBadRequest, "Current and new password required")
		return
	}

	// Get user to verify current password
	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify current password
	valid, err := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if err != nil || !valid {
		ErrorResponse(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Update password
	if err := h.db.UpdateUser(r.Context(), userID, map[string]any{
		"password_hash": newHash,
	}); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	SuccessResponse(w, map[string]string{"status": "password changed"})
}
