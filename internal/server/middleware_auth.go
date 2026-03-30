package server

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/auth"
)

// contextKey type for context values.
type contextKey string

const (
	userIDKey    contextKey = "user_id"
	usernameKey  contextKey = "username"
	rolesKey     contextKey = "roles"
	deviceIDKey  contextKey = "device_id"
)

// AuthMiddleware handles authentication for HTTP requests.
type AuthMiddleware struct {
	authService    *auth.Service
	apiTokenService *auth.APITokenService
	rbac           *auth.RBAC
	jwtSecret      []byte
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(authService *auth.Service, apiTokenService *auth.APITokenService, rbac *auth.RBAC, jwtSecret []byte) *AuthMiddleware {
	return &AuthMiddleware{
		authService:     authService,
		apiTokenService: apiTokenService,
		rbac:            rbac,
		jwtSecret:       jwtSecret,
	}
}

// Authenticate is middleware that extracts and validates authentication.
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try JWT Bearer token first
		if token := extractBearerToken(r); token != "" {
			claims, err := am.authService.ValidateAccessToken(token)
			if err == nil {
				// Valid JWT - set context
				ctx := am.setUserContext(r.Context(), claims.UserID, claims.Username, claims.Roles, claims.DeviceID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try API token
		if token := extractAPIToken(r); token != "" {
			user, permissions, err := am.apiTokenService.ValidateToken(r.Context(), token)
			if err == nil {
				// Valid API token - set context with permissions as roles
				ctx := am.setAPIUserContext(r.Context(), user.ID, user.Username, permissions)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try session cookie for Web UI
		if sessionID, err := r.Cookie("session_id"); err == nil && sessionID.Value != "" {
			// Session cookie handling would be implemented here
			// For now, continue as unauthenticated
		}

		// No valid auth - continue as unauthenticated (routes can check if needed)
		next.ServeHTTP(w, r)
	})
}

// RequireAuth ensures the request is authenticated.
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromRequest(r)
		if userID == "" {
			am.writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePermission creates middleware that requires a specific permission.
func (am *AuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserIDFromRequest(r)
			if userID == "" {
				am.writeError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Get scope from context or default to own
			scope := "own"
			if r.URL.Query().Get("scope") != "" {
				scope = r.URL.Query().Get("scope")
			}

			if err := am.rbac.Authorize(r.Context(), userID, resource, action, scope); err != nil {
				am.writeError(w, http.StatusForbidden, "Permission denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin requires the user to be an admin.
func (am *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromRequest(r)
		if userID == "" {
			am.writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		isAdmin, err := am.rbac.IsAdmin(r.Context(), userID)
		if err != nil || !isAdmin {
			am.writeError(w, http.StatusForbidden, "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth extracts auth info if present but doesn't require it.
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return am.Authenticate(next)
}

// extractBearerToken extracts the Bearer token from Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimPrefix(auth, prefix)
}

// extractAPIToken extracts the API token from Authorization header.
func extractAPIToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Token "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimPrefix(auth, prefix)
}

// setUserContext sets user info in the request context.
func (am *AuthMiddleware) setUserContext(ctx context.Context, userID, username string, roles []string, deviceID string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	ctx = context.WithValue(ctx, rolesKey, roles)
	ctx = context.WithValue(ctx, deviceIDKey, deviceID)
	return ctx
}

// setAPIUserContext sets API user info in the request context.
func (am *AuthMiddleware) setAPIUserContext(ctx context.Context, userID, username string, permissions []string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	// API permissions are stored as roles for compatibility
	ctx = context.WithValue(ctx, rolesKey, permissions)
	return ctx
}

// writeError writes an error response.
func (am *AuthMiddleware) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// GetUserIDFromRequest extracts the user ID from the request context.
func GetUserIDFromRequest(r *http.Request) string {
	if v := r.Context().Value(userIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUsernameFromRequest extracts the username from the request context.
func GetUsernameFromRequest(r *http.Request) string {
	if v := r.Context().Value(usernameKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetRolesFromRequest extracts the roles from the request context.
func GetRolesFromRequest(r *http.Request) []string {
	if v := r.Context().Value(rolesKey); v != nil {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}

// GetDeviceIDFromRequest extracts the device ID from the request context.
func GetDeviceIDFromRequest(r *http.Request) string {
	if v := r.Context().Value(deviceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// IsAuthenticated checks if the request is authenticated.
func IsAuthenticated(r *http.Request) bool {
	return GetUserIDFromRequest(r) != ""
}

// IsAdmin checks if the request is from an admin user.
func IsAdmin(r *http.Request) bool {
	roles := GetRolesFromRequest(r)
	return slices.Contains(roles, "admin")
}

// PublicRoute is middleware that marks a route as public (no auth required).
// This is useful for documentation/logging but doesn't change behavior.
func PublicRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just pass through - this is a marker middleware
		next.ServeHTTP(w, r)
	})
}

// AuthInfo holds authentication information extracted from a request.
type AuthInfo struct {
	UserID   string
	Username string
	Roles    []string
	DeviceID string
	IsAPIKey bool
}

// GetAuthInfo extracts full auth info from a request.
func GetAuthInfo(r *http.Request) *AuthInfo {
	return &AuthInfo{
		UserID:   GetUserIDFromRequest(r),
		Username: GetUsernameFromRequest(r),
		Roles:    GetRolesFromRequest(r),
		DeviceID: GetDeviceIDFromRequest(r),
	}
}

// CookieAuth creates a cookie-based auth handler for Web UI.
type CookieAuth struct {
	authService *auth.Service
	cookieName  string
	secure      bool
	maxAge      int
}

// NewCookieAuth creates a new cookie auth handler.
func NewCookieAuth(authService *auth.Service, cookieName string, secure bool, maxAge int) *CookieAuth {
	return &CookieAuth{
		authService: authService,
		cookieName:  cookieName,
		secure:      secure,
		maxAge:      maxAge,
	}
}

// SetAuthCookie sets the authentication cookie.
func (ca *CookieAuth) SetAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     ca.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   ca.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   ca.maxAge,
	})
}

// ClearAuthCookie clears the authentication cookie.
func (ca *CookieAuth) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     ca.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   ca.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
