package server

import (
	"net/http"

	"github.com/vaultdrift/vaultdrift/internal/auth"
)

// AuthHandler handles authentication API requests.
type AuthHandler struct {
	authSvc *auth.Service
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authSvc: authService}
}

// RegisterRoutes registers the auth routes.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	// Login
	mux.HandleFunc("POST /api/v1/auth/login", h.login)

	// TOTP verification (second factor)
	mux.HandleFunc("POST /api/v1/auth/totp/verify", h.verifyTOTP)

	// Logout (requires auth)
	mux.Handle("POST /api/v1/auth/logout", h.requireAuth(http.HandlerFunc(h.logout)))

	// Refresh token
	mux.HandleFunc("POST /api/v1/auth/refresh", h.refresh)
}

// loginRequest represents a login request.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login handles user login.
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		ErrorResponse(w, http.StatusBadRequest, "Username and password required")
		return
	}

	// Perform login
	result, err := h.authSvc.Login(r.Context(), req.Username, req.Password, "", "web", r.RemoteAddr, r.UserAgent())
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			ErrorResponse(w, http.StatusUnauthorized, "Invalid username or password")
		case auth.ErrUserDisabled:
			ErrorResponse(w, http.StatusForbidden, "Account is disabled")
		case auth.ErrUserLocked:
			ErrorResponse(w, http.StatusForbidden, "Account is locked")
		case auth.ErrPasswordChangeRequired:
			ErrorResponse(w, http.StatusForbidden, "Password change required")
		default:
			InternalErrorResponse(w, err)
		}
		return
	}

	// Check if TOTP is required
	if result.RequiresTOTP {
		SuccessResponse(w, map[string]any{
			"requires_totp": true,
			"totp_session":  result.TOTPSession,
		})
		return
	}

	SuccessResponse(w, map[string]any{
		"token":         result.Tokens.AccessToken,
		"refresh_token": result.Tokens.RefreshToken,
		"expires_at":    result.Tokens.ExpiresAt,
		"session_id":    result.SessionID,
		"username":      req.Username,
	})
}

// totpVerifyRequest represents a TOTP verification request.
type totpVerifyRequest struct {
	Session string `json:"session"`
	Code    string `json:"code"`
}

// verifyTOTP handles TOTP code verification during two-factor login.
func (h *AuthHandler) verifyTOTP(w http.ResponseWriter, r *http.Request) {
	var req totpVerifyRequest
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Session == "" || req.Code == "" {
		ErrorResponse(w, http.StatusBadRequest, "Session and code required")
		return
	}

	result, err := h.authSvc.LoginWithTOTP(r.Context(), req.Session, req.Code, "", "web", r.RemoteAddr, r.UserAgent())
	if err != nil {
		switch err {
		case auth.ErrInvalidSession:
			ErrorResponse(w, http.StatusUnauthorized, "Invalid or expired TOTP session")
		case auth.ErrInvalidTOTP:
			ErrorResponse(w, http.StatusUnauthorized, "Invalid TOTP code")
		default:
			InternalErrorResponse(w, err)
		}
		return
	}

	SuccessResponse(w, map[string]any{
		"token":         result.Tokens.AccessToken,
		"refresh_token": result.Tokens.RefreshToken,
		"expires_at":    result.Tokens.ExpiresAt,
		"session_id":    result.SessionID,
	})
}

// logout handles user logout.
func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	// Get token from header
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Validate token to get session info
	claims, err := h.authSvc.ValidateAccessToken(token)
	if err != nil {
		ErrorResponse(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	// Delete session using SessionID from claims
	if err := h.authSvc.Logout(r.Context(), claims.SessionID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	SuccessResponse(w, map[string]string{"status": "logged out"})
}

// refresh handles token refresh.
func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		ErrorResponse(w, http.StatusBadRequest, "Refresh token required")
		return
	}

	tokens, err := h.authSvc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case auth.ErrInvalidSession:
			ErrorResponse(w, http.StatusUnauthorized, "Invalid session")
		case auth.ErrSessionExpired:
			ErrorResponse(w, http.StatusUnauthorized, "Session expired")
		default:
			InternalErrorResponse(w, err)
		}
		return
	}

	SuccessResponse(w, map[string]any{
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_at":    tokens.ExpiresAt,
	})
}

// requireAuth middleware checks for valid authentication.
func (h *AuthHandler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			ErrorResponse(w, http.StatusUnauthorized, "Authorization required")
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		_, err := h.authSvc.ValidateAccessToken(token)
		if err != nil {
			ErrorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
