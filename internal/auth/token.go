package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/crypto"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/util"
)

// API token constants.
const (
	APITokenPrefix = "vd_" // VaultDrift API token prefix
	APITokenLength = 48    // Length of the raw token (excluding prefix)
)

var (
	// ErrInvalidAPIToken is returned when the API token is invalid.
	ErrInvalidAPIToken = errors.New("invalid API token")
	// ErrAPITokenExpired is returned when the API token has expired.
	ErrAPITokenExpired = errors.New("API token expired")
	// ErrAPITokenRevoked is returned when the API token has been revoked.
	ErrAPITokenRevoked = errors.New("API token revoked")
)

// APITokenService handles API token operations.
type APITokenService struct {
	db *db.Manager
}

// NewAPITokenService creates a new API token service.
func NewAPITokenService(db *db.Manager) *APITokenService {
	return &APITokenService{db: db}
}

// GenerateTokenResult contains the generated token information.
type GenerateTokenResult struct {
	Token       string   // The raw token (returned once)
	TokenID     string   // The token ID (for revocation)
	Name        string   // Token name
	UserID      string   // Owner user ID
	Permissions []string // Scoped permissions
	ExpiresAt   *time.Time
}

// GenerateToken creates a new API token with scoped permissions.
// The raw token is only returned once and must be stored securely by the client.
func (s *APITokenService) GenerateToken(ctx context.Context, userID, name string, permissions []string) (*GenerateTokenResult, error) {
	// Validate input
	if name == "" {
		return nil, fmt.Errorf("token name is required")
	}

	// Validate permissions are valid for this user
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get user's available permissions from role
	userPerms, err := s.getUserAvailablePermissions(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Validate requested permissions are subset of user's permissions
	validPerms := s.filterValidPermissions(permissions, userPerms)
	if len(validPerms) == 0 && len(permissions) > 0 {
		return nil, fmt.Errorf("no valid permissions requested")
	}

	// Generate token ID
	tokenID := util.GenerateUUID()

	// Generate raw token: vd_<random>
	rawToken, err := s.generateRawToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token for storage (SHA-256)
	tokenHash := s.hashToken(rawToken)

	// Store token in database
	tokenRecord := &db.APIToken{
		ID:          tokenID,
		UserID:      userID,
		Name:        name,
		TokenHash:   tokenHash,
		Permissions: validPerms,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.db.CreateAPIToken(ctx, tokenRecord); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return &GenerateTokenResult{
		Token:       rawToken,
		TokenID:     tokenID,
		Name:        name,
		UserID:      userID,
		Permissions: validPerms,
		ExpiresAt:   nil, // No expiry by default
	}, nil
}

// ValidateToken validates an API token and returns the associated user and permissions.
func (s *APITokenService) ValidateToken(ctx context.Context, token string) (*db.User, []string, error) {
	// Validate token format
	if !strings.HasPrefix(token, APITokenPrefix) {
		return nil, nil, ErrInvalidAPIToken
	}

	// Hash the token
	tokenHash := s.hashToken(token)

	// Look up token in database
	tokenRecord, err := s.db.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, ErrInvalidAPIToken
	}

	// Check if expired
	if tokenRecord.ExpiresAt != nil && time.Now().UTC().After(*tokenRecord.ExpiresAt) {
		return nil, nil, ErrAPITokenExpired
	}

	// Get user
	user, err := s.db.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("token user not found: %w", err)
	}

	// Update last used time
	now := time.Now().UTC()
	if err := s.db.UpdateAPITokenLastUsed(ctx, tokenRecord.ID, now); err != nil {
		// Non-fatal, just log
		fmt.Printf("Failed to update API token last used: %v\n", err)
	}

	return user, tokenRecord.Permissions, nil
}

// RevokeToken revokes an API token by ID.
func (s *APITokenService) RevokeToken(ctx context.Context, tokenID string) error {
	return s.db.DeleteAPIToken(ctx, tokenID)
}

// RevokeAllUserTokens revokes all API tokens for a user.
func (s *APITokenService) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return s.db.DeleteAPITokensByUser(ctx, userID)
}

// ListUserTokens lists all API tokens for a user (without the hashes).
func (s *APITokenService) ListUserTokens(ctx context.Context, userID string) ([]*db.APIToken, error) {
	return s.db.ListAPITokensByUser(ctx, userID)
}

// generateRawToken generates a new random API token.
func (s *APITokenService) generateRawToken() (string, error) {
	// Generate 32 random bytes
	randomBytes, err := crypto.RandomBytes(32)
	if err != nil {
		return "", err
	}

	// Encode as base64url (without padding) to get ~43 chars
	tokenPart := util.Base64URLEncode(randomBytes)

	// Ensure minimum length
	if len(tokenPart) < APITokenLength-len(APITokenPrefix) {
		return "", fmt.Errorf("token generation failed")
	}

	// Take first APITokenLength characters after prefix
	if len(tokenPart) > APITokenLength-len(APITokenPrefix) {
		tokenPart = tokenPart[:APITokenLength-len(APITokenPrefix)]
	}

	return APITokenPrefix + tokenPart, nil
}

// hashToken creates a SHA-256 hash of the token for storage.
func (s *APITokenService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// getUserAvailablePermissions returns the permissions available to a user based on their role.
func (s *APITokenService) getUserAvailablePermissions(ctx context.Context, user *db.User) ([]string, error) {
	// Map roles to available permissions
	rolePerms := map[string][]string{
		"admin": {
			"file:read:all", "file:write:all", "file:delete:all",
			"folder:read:all", "folder:write:all", "folder:delete:all",
			"share:read:all", "share:write:all", "share:delete:all",
			"user:read:all", "user:write:all", "user:delete:all",
			"system:manage:all",
		},
		"user": {
			"file:read:own", "file:write:own", "file:delete:own",
			"file:read:group",
			"folder:read:own", "folder:write:own", "folder:delete:own",
			"folder:read:group",
			"share:read:own", "share:write:own", "share:delete:own",
			"share:read:group",
			"user:read:own", "user:write:own",
		},
		"guest": {
			"file:read:group",
			"folder:read:group",
		},
	}

	perms, ok := rolePerms[user.Role]
	if !ok {
		return []string{}, nil
	}

	return perms, nil
}

// filterValidPermissions filters requested permissions to only include valid ones.
func (s *APITokenService) filterValidPermissions(requested, available []string) []string {
	availableSet := make(map[string]bool)
	for _, p := range available {
		availableSet[p] = true
	}

	// If no specific permissions requested, use all available
	if len(requested) == 0 {
		return available
	}

	// Filter to only valid permissions
	valid := make([]string, 0, len(requested))
	for _, p := range requested {
		if availableSet[p] {
			valid = append(valid, p)
		}
	}

	return valid
}

// ParsePermission parses a permission string in format "resource:action:scope".
func ParsePermission(perm string) (resource, action, scope string, err error) {
	parts := strings.Split(perm, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid permission format: %s", perm)
	}
	return parts[0], parts[1], parts[2], nil
}

// FormatPermission formats a permission as "resource:action:scope".
func FormatPermission(resource, action, scope string) string {
	return fmt.Sprintf("%s:%s:%s", resource, action, scope)
}

// HasPermission checks if a permission list contains a specific permission.
func HasPermission(permissions []string, resource, action, scope string) bool {
	requested := FormatPermission(resource, action, scope)

	for _, p := range permissions {
		if p == requested {
			return true
		}
		// Check wildcards
		if strings.HasSuffix(p, ":*") || strings.HasSuffix(p, ":manage:*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(requested, prefix) {
				return true
			}
		}
	}

	return false
}
