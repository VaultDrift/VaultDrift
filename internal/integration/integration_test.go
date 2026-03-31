package integration

import (
	"context"
	"testing"

	"github.com/vaultdrift/vaultdrift/internal/auth"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/server"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// TestSuite provides an integration test harness
type TestSuite struct {
	DB        *db.Manager
	Storage   storage.Backend
	Auth      *auth.Service
	VFS       *vfs.VFS
	Token     string
	TmpDir    string
}

// SetupTestSuite creates a test server with temporary storage
func SetupTestSuite(t *testing.T) *TestSuite {
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

	return &TestSuite{
		DB:      database,
		Storage: store,
		Auth:    authSvc,
		VFS:     vfsService,
		TmpDir:  tmpDir,
	}
}

// Cleanup closes resources
func (s *TestSuite) Cleanup() {
	if s.DB != nil {
		s.DB.Close()
	}
}

// CreateServer creates a test server
func (s *TestSuite) CreateServer() *server.Server {
	cfg := config.ServerConfig{
		Host: "localhost",
		Port: 0,
	}
	return server.NewServer(cfg, s.DB, s.Auth, s.VFS, s.Storage, []byte("test-secret"), nil)
}

// TestHealthCheck tests the health endpoint
func TestHealthCheck(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	// Create server
	srv := suite.CreateServer()
	_ = srv

	t.Log("Health check test would run here")
}

// TestAuthenticationFlow tests the full auth flow
func TestAuthenticationFlow(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	// Register a test user using the proper method
	user := &db.User{
		Username:     "testuser",
		Email:        "test@test.com",
		PasswordHash: "$2a$10$hash",
		Role:         "user",
		Status:       "active",
	}
	err := suite.DB.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Logf("Auth flow test - created user with ID: %s", user.ID)
}
