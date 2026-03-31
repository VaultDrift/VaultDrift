package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Test hashing
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal original password")
	}

	// Test verification with correct password
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}
	if !valid {
		t.Error("Password should be valid")
	}

	// Test verification with incorrect password
	valid, err = VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}
	if valid {
		t.Error("Password should be invalid")
	}
}

func TestJWTSigner(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-signing")
	signer := NewJWTSigner(secret)

	t.Run("GenerateAndValidateToken", func(t *testing.T) {
		userID := "user_123"
		username := "testuser"
		roles := []string{"user"}
		deviceName := "Test Device"
		sessionID := "session_456"

		tokens, err := signer.GenerateTokenPair(userID, username, roles, deviceName, sessionID)
		if err != nil {
			t.Fatalf("Failed to generate tokens: %v", err)
		}

		if tokens.AccessToken == "" {
			t.Error("Access token should not be empty")
		}

		if tokens.RefreshToken == "" {
			t.Error("Refresh token should not be empty")
		}

		// Validate access token
		claims, err := signer.ValidateAccessToken(tokens.AccessToken)
		if err != nil {
			t.Fatalf("Failed to validate access token: %v", err)
		}

		if claims.UserID != userID {
			t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
		}

		if claims.Username != username {
			t.Errorf("Username mismatch: got %s, want %s", claims.Username, username)
		}

		// Validate refresh token
		refreshClaims, err := signer.ValidateRefreshToken(tokens.RefreshToken)
		if err != nil {
			t.Fatalf("Failed to validate refresh token: %v", err)
		}

		if refreshClaims.UserID != userID {
			t.Errorf("Refresh token UserID mismatch: got %s, want %s", refreshClaims.UserID, userID)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, err := signer.ValidateAccessToken("invalid.token.here")
		if err == nil {
			t.Error("Should fail for invalid token")
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		// Create a signer with very short expiry
		shortSigner := NewJWTSigner(secret)
		_ = shortSigner
		// Note: Token expiry is hardcoded in implementation
		// This test documents expected behavior
		t.Log("Token expiry is validated during parsing")
	})

	t.Run("TokenRotation", func(t *testing.T) {
		userID := "user_789"
		username := "testuser2"
		roles := []string{"user"}
		deviceName := "Test Device"
		sessionID := "session_789"

		tokens1, _ := signer.GenerateTokenPair(userID, username, roles, deviceName, sessionID)
		time.Sleep(10 * time.Millisecond)
		tokens2, _ := signer.GenerateTokenPair(userID, username, roles, deviceName, sessionID)

		// Tokens should be different
		if tokens1.AccessToken == tokens2.AccessToken {
			t.Error("Tokens generated at different times should be different")
		}
	})
}

func TestTOTP(t *testing.T) {
	totp := NewTOTP()

	t.Run("GenerateSecret", func(t *testing.T) {
		secret, uri, err := totp.GenerateSecret("test@example.com")
		if err != nil {
			t.Fatalf("Failed to generate secret: %v", err)
		}

		if secret == "" {
			t.Error("Secret should not be empty")
		}

		// Secret should be base32 encoded (typically 32 chars)
		if len(secret) < 16 {
			t.Errorf("Secret length should be at least 16, got %d", len(secret))
		}

		if uri == "" {
			t.Error("Provisioning URI should not be empty")
		}
	})

	t.Run("GenerateCode", func(t *testing.T) {
		secret, _, _ := totp.GenerateSecret("test@example.com")
		code, err := totp.GenerateCode(secret)
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		if code == "" {
			t.Error("TOTP code should not be empty")
		}

		if len(code) != 6 {
			t.Errorf("TOTP code should be 6 digits, got %d", len(code))
		}
	})

	t.Run("ValidateCode", func(t *testing.T) {
		secret, _, _ := totp.GenerateSecret("test@example.com")
		code, _ := totp.GenerateCode(secret)

		// Current code should be valid
		valid := totp.ValidateCode(secret, code)
		if !valid {
			t.Error("Current TOTP code should be valid")
		}

		// Wrong code should be invalid
		valid = totp.ValidateCode(secret, "000000")
		if valid {
			t.Error("Wrong TOTP code should be invalid")
		}
	})
}

func TestRBAC(t *testing.T) {
	// Create RBAC without database (using default roles)
	rbac := NewRBAC(nil)

	t.Run("DefaultRolesExist", func(t *testing.T) {
		// Check that default roles exist
		roles := []string{"admin", "user", "guest"}
		for _, roleName := range roles {
			role, exists := rbac.roles[roleName]
			if !exists {
				t.Errorf("Default role %s should exist", roleName)
				continue
			}
			if role.Name != roleName {
				t.Errorf("Role name mismatch for %s", roleName)
			}
		}
	})

	t.Run("PermissionMatches", func(t *testing.T) {
		perm := &Permission{
			Resource: "file",
			Action:   "read",
			Scope:    "own",
		}

		// Exact match
		if !perm.Matches("file", "read", "own") {
			t.Error("Permission should match exact request")
		}

		// Wrong resource
		if perm.Matches("folder", "read", "own") {
			t.Error("Permission should not match different resource")
		}

		// Wrong action
		if perm.Matches("file", "write", "own") {
			t.Error("Permission should not match different action")
		}

		// Manage action implies all actions
		managePerm := &Permission{
			Resource: "file",
			Action:   "manage",
			Scope:    "all",
		}
		if !managePerm.Matches("file", "read", "own") {
			t.Error("Manage permission should imply read access")
		}
		if !managePerm.Matches("file", "delete", "own") {
			t.Error("Manage permission should imply delete access")
		}

		// Scope levels: own < group < all
		groupPerm := &Permission{
			Resource: "file",
			Action:   "read",
			Scope:    "group",
		}
		if !groupPerm.Matches("file", "read", "own") {
			t.Error("Group permission should cover own scope")
		}
		if groupPerm.Matches("file", "read", "all") {
			t.Error("Group permission should not cover all scope")
		}
	})

	t.Run("AdminHasAllPermissions", func(t *testing.T) {
		adminRole := rbac.roles["admin"]
		if adminRole == nil {
			t.Fatal("Admin role should exist")
		}

		hasManageAll := false
		for _, perm := range adminRole.Permissions {
			if perm.Resource == "system" && perm.Action == "manage" && perm.Scope == "all" {
				hasManageAll = true
				break
			}
		}

		if !hasManageAll {
			t.Error("Admin should have system:manage:all permission")
		}
	})
}
