// Package auth provides authentication and authorization functionality.
package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// RBAC errors.
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidResource  = errors.New("invalid resource")
	ErrInvalidAction    = errors.New("invalid action")
)

// Permission represents a specific authorization grant.
type Permission struct {
	Resource string // e.g., "file", "folder", "user", "system"
	Action   string // e.g., "read", "write", "delete", "manage"
	Scope    string // e.g., "own", "group", "all"
}

// Matches checks if this permission matches the requested operation.
func (p *Permission) Matches(resource, action, scope string) bool {
	// Resource must match exactly
	if p.Resource != resource {
		return false
	}

	// Action must match or be "manage" (manage implies all actions)
	if p.Action != action && p.Action != "manage" {
		return false
	}

	// Scope must be sufficient
	// own < group < all
	scopeLevels := map[string]int{"own": 1, "group": 2, "all": 3}
	requiredLevel := scopeLevels[scope]
	availableLevel := scopeLevels[p.Scope]

	return availableLevel >= requiredLevel
}

// Role represents a named set of permissions.
type Role struct {
	ID          string
	Name        string
	Description string
	IsSystem    bool
	Permissions []Permission
}

// RBAC handles role-based access control.
type RBAC struct {
	db    *db.Manager
	roles map[string]*Role        // in-memory cache
	perms map[string][]Permission // userID -> permissions cache
	mu    sync.RWMutex
}

// NewRBAC creates a new RBAC engine.
func NewRBAC(db *db.Manager) *RBAC {
	rbac := &RBAC{
		db:    db,
		roles: make(map[string]*Role),
		perms: make(map[string][]Permission),
	}

	// Seed default roles
	rbac.seedDefaultRoles()

	return rbac
}

// seedDefaultRoles creates the default system roles.
func (r *RBAC) seedDefaultRoles() {
	// Admin role: all resources, all actions, scope=all
	adminRole := &Role{
		ID:          "role_admin",
		Name:        "admin",
		Description: "Full system access",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: "file", Action: "manage", Scope: "all"},
			{Resource: "folder", Action: "manage", Scope: "all"},
			{Resource: "share", Action: "manage", Scope: "all"},
			{Resource: "user", Action: "manage", Scope: "all"},
			{Resource: "system", Action: "manage", Scope: "all"},
		},
	}

	// User role: file/folder/share (own), read (group), no user/system manage
	userRole := &Role{
		ID:          "role_user",
		Name:        "user",
		Description: "Standard user access",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: "file", Action: "read", Scope: "group"},
			{Resource: "file", Action: "write", Scope: "own"},
			{Resource: "file", Action: "delete", Scope: "own"},
			{Resource: "folder", Action: "read", Scope: "group"},
			{Resource: "folder", Action: "write", Scope: "own"},
			{Resource: "folder", Action: "delete", Scope: "own"},
			{Resource: "share", Action: "read", Scope: "group"},
			{Resource: "share", Action: "write", Scope: "own"},
			{Resource: "share", Action: "delete", Scope: "own"},
			{Resource: "user", Action: "read", Scope: "own"},
			{Resource: "user", Action: "write", Scope: "own"},
		},
	}

	// Guest role: file/folder read (group only)
	guestRole := &Role{
		ID:          "role_guest",
		Name:        "guest",
		Description: "Limited read-only access",
		IsSystem:    true,
		Permissions: []Permission{
			{Resource: "file", Action: "read", Scope: "group"},
			{Resource: "folder", Action: "read", Scope: "group"},
		},
	}

	r.roles["admin"] = adminRole
	r.roles["user"] = userRole
	r.roles["guest"] = guestRole
}

// GetRole returns a role by name.
func (r *RBAC) GetRole(name string) (*Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, ok := r.roles[name]
	return role, ok
}

// Authorize checks if a user has permission for an action on a resource.
func (r *RBAC) Authorize(ctx context.Context, userID, resource, action, scope string) error {
	// Validate inputs
	if resource == "" || action == "" {
		return ErrInvalidResource
	}

	if scope == "" {
		scope = "own" // default scope
	}

	// Get user's permissions from cache or database
	permissions, err := r.getUserPermissions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if any permission matches
	for _, perm := range permissions {
		if perm.Matches(resource, action, scope) {
			return nil
		}
	}

	return ErrPermissionDenied
}

// AuthorizeOwn checks if user can access their own resource.
func (r *RBAC) AuthorizeOwn(ctx context.Context, userID, resource, action string) error {
	return r.Authorize(ctx, userID, resource, action, "own")
}

// AuthorizeGroup checks if user can access group-shared resources.
func (r *RBAC) AuthorizeGroup(ctx context.Context, userID, resource, action string) error {
	return r.Authorize(ctx, userID, resource, action, "group")
}

// AuthorizeAny checks if user can access any resource (admin-level).
func (r *RBAC) AuthorizeAny(ctx context.Context, userID, resource, action string) error {
	return r.Authorize(ctx, userID, resource, action, "all")
}

// getUserPermissions retrieves permissions for a user.
func (r *RBAC) getUserPermissions(ctx context.Context, userID string) ([]Permission, error) {
	// Check cache first
	r.mu.RLock()
	cached, ok := r.perms[userID]
	r.mu.RUnlock()

	if ok {
		return cached, nil
	}

	// Get user from database
	user, err := r.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get role permissions
	r.mu.RLock()
	role, ok := r.roles[user.Role]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("role not found: %s", user.Role)
	}

	// Cache permissions
	perms := make([]Permission, len(role.Permissions))
	copy(perms, role.Permissions)

	r.mu.Lock()
	r.perms[userID] = perms
	r.mu.Unlock()

	return perms, nil
}

// InvalidateUserCache clears the permission cache for a user.
func (r *RBAC) InvalidateUserCache(userID string) {
	r.mu.Lock()
	delete(r.perms, userID)
	r.mu.Unlock()
}

// InvalidateAllCache clears the entire permission cache.
func (r *RBAC) InvalidateAllCache() {
	r.mu.Lock()
	r.perms = make(map[string][]Permission)
	r.mu.Unlock()
}

// HasRole checks if a user has a specific role.
func (r *RBAC) HasRole(ctx context.Context, userID, roleName string) (bool, error) {
	user, err := r.db.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	return user.Role == roleName, nil
}

// IsAdmin checks if a user is an admin.
func (r *RBAC) IsAdmin(ctx context.Context, userID string) (bool, error) {
	return r.HasRole(ctx, userID, "admin")
}

// MiddlewareFunc is the signature for RBAC middleware.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// HandlerFunc is the handler signature.
type HandlerFunc func(ctx context.Context) error

// RequirePermission creates middleware that enforces a permission.
func (r *RBAC) RequirePermission(resource, action string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			// Get userID from context
			userID := GetUserIDFromContext(ctx)
			if userID == "" {
				return ErrPermissionDenied
			}

			// Get scope from context (default to own)
			scope := GetScopeFromContext(ctx)
			if scope == "" {
				scope = "own"
			}

			// Authorize
			if err := r.Authorize(ctx, userID, resource, action, scope); err != nil {
				return err
			}

			return next(ctx)
		}
	}
}

// RequireRole creates middleware that enforces a role.
func (r *RBAC) RequireRole(roleName string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			userID := GetUserIDFromContext(ctx)
			if userID == "" {
				return ErrPermissionDenied
			}

			hasRole, err := r.HasRole(ctx, userID, roleName)
			if err != nil {
				return err
			}

			if !hasRole {
				return ErrPermissionDenied
			}

			return next(ctx)
		}
	}
}

// RequireAdmin is a shortcut for requiring admin role.
func (r *RBAC) RequireAdmin() MiddlewareFunc {
	return r.RequireRole("admin")
}

// Context keys for auth information.
type contextKey string

const (
	userIDKey   contextKey = "user_id"
	rolesKey    contextKey = "roles"
	scopeKey    contextKey = "scope"
	deviceIDKey contextKey = "device_id"
)

// WithUserID adds user ID to context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext extracts user ID from context.
func GetUserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// WithRoles adds roles to context.
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, rolesKey, roles)
}

// GetRolesFromContext extracts roles from context.
func GetRolesFromContext(ctx context.Context) []string {
	if v, ok := ctx.Value(rolesKey).([]string); ok {
		return v
	}
	return nil
}

// WithScope adds scope to context.
func WithScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, scopeKey, scope)
}

// GetScopeFromContext extracts scope from context.
func GetScopeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(scopeKey).(string); ok {
		return v
	}
	return ""
}

// WithDeviceID adds device ID to context.
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, deviceIDKey, deviceID)
}

// GetDeviceIDFromContext extracts device ID from context.
func GetDeviceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(deviceIDKey).(string); ok {
		return v
	}
	return ""
}

// PermissionChecker provides a fluent API for checking permissions.
type PermissionChecker struct {
	rbac   *RBAC
	ctx    context.Context
	userID string
}

// ForUser creates a permission checker for a user.
func (r *RBAC) ForUser(ctx context.Context, userID string) *PermissionChecker {
	return &PermissionChecker{
		rbac:   r,
		ctx:    ctx,
		userID: userID,
	}
}

// Can checks if the user can perform an action on a resource.
func (pc *PermissionChecker) Can(action, resource string) bool {
	return pc.rbac.Authorize(pc.ctx, pc.userID, resource, action, "own") == nil
}

// CanInScope checks if the user can perform an action with a specific scope.
func (pc *PermissionChecker) CanInScope(action, resource, scope string) bool {
	return pc.rbac.Authorize(pc.ctx, pc.userID, resource, action, scope) == nil
}

// Is checks if the user has a specific role.
func (pc *PermissionChecker) Is(role string) bool {
	hasRole, _ := pc.rbac.HasRole(pc.ctx, pc.userID, role)
	return hasRole
}

// ListPermissions returns all permissions for a user.
func (r *RBAC) ListPermissions(ctx context.Context, userID string) ([]Permission, error) {
	return r.getUserPermissions(ctx, userID)
}

// AddRole creates a new custom role.
func (r *RBAC) AddRole(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.roles[role.Name]; exists {
		return fmt.Errorf("role already exists: %s", role.Name)
	}

	r.roles[role.Name] = role
	return nil
}

// UpdateRole updates a role's permissions.
func (r *RBAC) UpdateRole(roleName string, permissions []Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, ok := r.roles[roleName]
	if !ok {
		return fmt.Errorf("role not found: %s", roleName)
	}

	if role.IsSystem {
		return fmt.Errorf("cannot modify system role: %s", roleName)
	}

	role.Permissions = permissions

	// Invalidate all permission caches since role changed
	r.perms = make(map[string][]Permission)

	return nil
}

// CleanupCachePeriodically runs a background goroutine to clean up expired cache entries.
func (r *RBAC) CleanupCachePeriodically(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			r.InvalidateAllCache()
		}
	}()
}
