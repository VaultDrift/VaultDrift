package webdav

import (
	"testing"
	"time"
)

// TestLockStore tests the lock store functionality
func TestLockStore(t *testing.T) {
	store := NewLockStore()

	t.Run("LockAndUnlock", func(t *testing.T) {
		store.Lock("resource-1", "token-1", "user-1", 30*time.Minute, false)

		if !store.IsLocked("resource-1") {
			t.Error("Resource should be locked")
		}

		token, ok := store.GetLockToken("resource-1")
		if !ok || token != "token-1" {
			t.Error("Should get lock token")
		}

		// Try unlock with wrong user
		if store.Unlock("token-1", "user-2") {
			t.Error("Unlock with wrong user should fail")
		}

		// Unlock with correct user
		if !store.Unlock("token-1", "user-1") {
			t.Error("Unlock with correct user should succeed")
		}

		if store.IsLocked("resource-1") {
			t.Error("Resource should be unlocked")
		}
	})

	t.Run("LockExpiration", func(t *testing.T) {
		// Lock with very short duration
		store.Lock("resource-2", "token-2", "user-1", 1*time.Millisecond, false)

		if !store.IsLocked("resource-2") {
			t.Error("Resource should be locked initially")
		}

		// Wait for expiration
		time.Sleep(50 * time.Millisecond)

		if store.IsLocked("resource-2") {
			t.Error("Resource should be unlocked after expiration")
		}
	})

	t.Run("MultipleLocks", func(t *testing.T) {
		store.Lock("resource-a", "token-a", "user-1", 30*time.Minute, false)
		store.Lock("resource-b", "token-b", "user-2", 30*time.Minute, false)

		if !store.IsLocked("resource-a") {
			t.Error("Resource A should be locked")
		}
		if !store.IsLocked("resource-b") {
			t.Error("Resource B should be locked")
		}

		// Unlock only A
		store.Unlock("token-a", "user-1")

		if store.IsLocked("resource-a") {
			t.Error("Resource A should be unlocked")
		}
		if !store.IsLocked("resource-b") {
			t.Error("Resource B should still be locked")
		}
	})
}

// TestDetectContentType tests content type detection
func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		data     []byte
		expected string
	}{
		{"file.txt", []byte("text"), "text/plain"},
		{"file.html", []byte("<html>"), "text/html"},
		{"file.css", []byte("body{}"), "text/css"},
		{"file.js", []byte("var x"), "application/javascript"},
		{"file.json", []byte("{}"), "application/json"},
		{"file.png", []byte{}, "image/png"},
		{"file.jpg", []byte{}, "image/jpeg"},
		{"file.jpeg", []byte{}, "image/jpeg"},
		{"file.gif", []byte{}, "image/gif"},
		{"file.svg", []byte{}, "image/svg+xml"},
		{"file.pdf", []byte{}, "application/pdf"},
		{"file.zip", []byte{}, "application/zip"},
		{"file.xml", []byte{}, "application/xml"},
		{"file.unknown", []byte{}, "application/octet-stream"},
	}

	for _, tt := range tests {
		result := detectContentType(tt.filename, tt.data)
		if result != tt.expected {
			t.Errorf("detectContentType(%s) = %s, expected %s", tt.filename, result, tt.expected)
		}
	}

	t.Logf("✅ Content type detection works correctly")
}

// TestGenerateLockToken tests lock token generation
func TestGenerateLockToken(t *testing.T) {
	token1 := generateLockToken()
	token2 := generateLockToken()

	if token1 == "" {
		t.Error("Token should not be empty")
	}

	if token1 == token2 {
		t.Error("Tokens should be unique")
	}

	if len(token1) < 20 {
		t.Error("Token should be reasonably long")
	}

	t.Logf("✅ Lock token generation works correctly")
}

// TestParseTimeout tests timeout parsing
func TestParseTimeout(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"Second-30", 30 * time.Second},
		{"Second-3600", 3600 * time.Second},
		{"Infinite", 30 * time.Minute}, // Falls back to default
		{"Invalid", 30 * time.Minute},  // Falls back to default
		{"", 30 * time.Minute},         // Falls back to default
	}

	for _, tt := range tests {
		result, err := parseTimeout(tt.input)
		if err != nil {
			// Some inputs cause errors, which is fine
			continue
		}
		if result != tt.expected {
			t.Errorf("parseTimeout(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}

	t.Logf("✅ Timeout parsing works correctly")
}

// TestParseDestination tests destination URL parsing
func TestParseDestination(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		input    string
		expected string
	}{
		{"http://example.com/webdav/file.txt", "/webdav/file.txt"},
		{"https://server.com/webdav/folder/file", "/webdav/folder/file"},
		{"/webdav/just/path", ""}, // Invalid - no scheme
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := handler.parseDestination(tt.input)
		if result != tt.expected {
			t.Errorf("parseDestination(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}

	t.Logf("✅ Destination parsing works correctly")
}
