package auth

import (
	"testing"
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
}
