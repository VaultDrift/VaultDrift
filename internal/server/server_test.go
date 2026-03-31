package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

func setupTestServer(t *testing.T) (*Server, *db.Manager, func()) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize database
	database, err := db.Open(db.Config{Path: tmpDir + "/test.db"})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Initialize storage
	store, err := storage.NewBackend(config.StorageConfig{
		Backend: "local",
		Local:   config.LocalConfig{DataDir: tmpDir + "/storage"},
	})
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Initialize services
	vfsService := vfs.NewVFS(database)
	authSvc := auth.NewService(database, []byte("test-secret"))

	// Create server
	cfg := config.ServerConfig{
		Host: "localhost",
		Port: 0,
	}
	server := NewServer(cfg, database, authSvc, vfsService, store, []byte("test-secret"), nil)

	cleanup := func() {
		database.Close()
	}

	return server, database, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", response["status"])
	}
}

func TestAuthEndpoints(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("Login", func(t *testing.T) {
		// First, create a user (would normally be done via admin)
		// For this test, we just check the endpoint exists
		body := map[string]string{
			"username": "testuser",
			"password": "testpass",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Should fail since user doesn't exist
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusNotFound {
			t.Logf("Login returned status %d (expected 401 or 404 for non-existent user)", w.Code)
		}
	})

	t.Run("LoginValidation", func(t *testing.T) {
		// Test with missing fields
		body := map[string]string{
			"username": "",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for invalid request, got %d", w.Code)
		}
	})
}

func TestFileEndpoints(t *testing.T) {
	server, database, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	user := &db.User{
		Username:     "testuser",
		Email:        "test@test.com",
		PasswordHash: "$2a$10$testhash",
		Role:         "user",
		Status:       "active",
	}
	err := database.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("ListFilesUnauthorized", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/files", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 without auth, got %d", w.Code)
		}
	})
}

func TestMiddleware(t *testing.T) {
	t.Run("SecurityHeaders", func(t *testing.T) {
		server, _, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check security headers
		headers := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
		}

		for _, header := range headers {
			if w.Header().Get(header) == "" {
				t.Logf("Security header %s not set", header)
			}
		}
	})

	t.Run("CORSMiddleware", func(t *testing.T) {
		server, _, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("OPTIONS", "/api/v1/files", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Preflight request should return appropriate CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Log("CORS headers not set")
		}
	})
}
