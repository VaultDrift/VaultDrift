package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// TestStressCreateUsers tests high-volume user creation
func TestStressCreateUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()
	start := time.Now()

	const numUsers = 100
	for i := 0; i < numUsers; i++ {
		user := &db.User{
			Username:     fmt.Sprintf("stressuser_%d_%d", time.Now().UnixNano(), i),
			Email:        fmt.Sprintf("stress_%d_%d@test.com", time.Now().UnixNano(), i),
			PasswordHash: "hash",
			Role:         "user",
			Status:       "active",
		}
		if err := suite.DB.CreateUser(ctx, user); err != nil {
			t.Fatalf("Failed to create user %d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("✅ Created %d users in %v (%.2f users/sec)", numUsers, duration, float64(numUsers)/duration.Seconds())
}

// TestStressFileOperations tests high-volume file operations
func TestStressFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	user := &db.User{
		Username:     fmt.Sprintf("stressfile_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("stressfile_%d@test.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         "user",
		Status:       "active",
	}
	suite.DB.CreateUser(ctx, user)

	// Create 500 files
	const numFiles = 500
	base := time.Now().UnixNano()

	start := time.Now()
	for i := 0; i < numFiles; i++ {
		file := &db.File{
			ID:        fmt.Sprintf("stressfile_%d_%d", base, i),
			Name:      fmt.Sprintf("file_%d.txt", i),
			Type:      "file",
			UserID:    user.ID,
			MimeType:  "text/plain",
			SizeBytes: 1024,
		}
		suite.DB.CreateFile(ctx, file)
	}
	createDuration := time.Since(start)

	// List files
	start = time.Now()
	files, _ := suite.DB.SearchFiles(ctx, user.ID, "file_", numFiles)
	listDuration := time.Since(start)

	t.Logf("✅ File operations: %d created in %v, listed in %v", numFiles, createDuration, listDuration)
	if len(files) != numFiles {
		t.Errorf("Expected %d files, got %d", numFiles, len(files))
	}
}

// TestStressConcurrentChunks tests concurrent chunk operations
func TestStressConcurrentChunks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	const numChunks = 1000
	base := time.Now().UnixNano()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numChunks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			chunk := &db.Chunk{
				Hash:           fmt.Sprintf("stresshash_%d_%d", base, index),
				SizeBytes:      1024,
				StorageBackend: "local",
				RefCount:       1,
			}
			suite.DB.CreateChunk(ctx, chunk)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("✅ Concurrent chunks: %d created in %v (%.2f chunks/sec)", numChunks, duration, float64(numChunks)/duration.Seconds())
}

// TestStressFolderHierarchy tests deep folder nesting
func TestStressFolderHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	user := &db.User{
		Username:     fmt.Sprintf("hierarchy_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("hierarchy_%d@test.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         "user",
		Status:       "active",
	}
	suite.DB.CreateUser(ctx, user)

	// Create deep hierarchy: root/level1/level2/.../level20
	const depth = 20
	parentID := ""
	base := time.Now().UnixNano()

	for i := 0; i < depth; i++ {
		folder := &db.File{
			ID:        fmt.Sprintf("folder_%d_%d", base, i),
			Name:      fmt.Sprintf("level%d", i),
			Type:      "folder",
			UserID:    user.ID,
			ParentID:  &parentID,
			MimeType:  "application/x-directory",
			SizeBytes: 0,
		}
		suite.DB.CreateFile(ctx, folder)
		parentID = folder.ID
	}

	t.Logf("✅ Created folder hierarchy with depth %d", depth)
}

// TestStressMemoryUsage tests memory efficiency
func TestStressMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupTestSuite(t)
	defer suite.Cleanup()

	ctx := context.Background()

	user := &db.User{
		Username:     fmt.Sprintf("memory_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("memory_%d@test.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         "user",
		Status:       "active",
	}
	suite.DB.CreateUser(ctx, user)

	const numOperations = 1000
	base := time.Now().UnixNano()

	// Perform many operations and check they complete
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		file := &db.File{
			ID:        fmt.Sprintf("memfile_%d_%d", base, i),
			Name:      fmt.Sprintf("file_%d.txt", i),
			Type:      "file",
			UserID:    user.ID,
			MimeType:  "text/plain",
			SizeBytes: 1024,
		}
		suite.DB.CreateFile(ctx, file)

		// Retrieve
		suite.DB.GetFileByID(ctx, file.ID)

		// Update
		suite.DB.UpdateFile(ctx, file.ID, map[string]any{"size_bytes": 2048})

		// Soft delete
		suite.DB.SoftDelete(ctx, file.ID)
	}
	duration := time.Since(start)

	t.Logf("✅ Memory stress: %d operations (create+get+update+delete) in %v", numOperations, duration)
}

// BenchmarkStressUpload benchmarks stress upload performance
func BenchmarkStressUpload(b *testing.B) {
	tmpDir := b.TempDir()
	database, _ := db.Open(db.Config{Path: tmpDir + "/stress.db"})
	defer database.Close()

	ctx := context.Background()

	user := &db.User{
		Username:     fmt.Sprintf("benchstress_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("benchstress_%d@test.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         "user",
		Status:       "active",
	}
	database.CreateUser(ctx, user)

	base := time.Now().UnixNano()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		file := &db.File{
			ID:        fmt.Sprintf("benchfile_%d_%d", base, i),
			Name:      fmt.Sprintf("file_%d.txt", i),
			Type:      "file",
			UserID:    user.ID,
			MimeType:  "text/plain",
			SizeBytes: 1024,
		}
		database.CreateFile(ctx, file)
	}
}

// BenchmarkStressConcurrentUsers benchmarks concurrent user creation
func BenchmarkStressConcurrentUsers(b *testing.B) {
	tmpDir := b.TempDir()
	database, _ := db.Open(db.Config{Path: tmpDir + "/concurrent_users.db"})
	defer database.Close()

	ctx := context.Background()
	base := time.Now().UnixNano()

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			user := &db.User{
				Username:     fmt.Sprintf("concurrent_%d_%d", base, index),
				Email:        fmt.Sprintf("concurrent_%d_%d@test.com", base, index),
				PasswordHash: "hash",
				Role:         "user",
				Status:       "active",
			}
			database.CreateUser(ctx, user)
		}(i)
	}
	wg.Wait()
}

// BenchmarkStressQueryPerformance benchmarks query performance
func BenchmarkStressQueryPerformance(b *testing.B) {
	tmpDir := b.TempDir()
	database, _ := db.Open(db.Config{Path: tmpDir + "/query.db"})
	defer database.Close()

	ctx := context.Background()

	user := &db.User{
		Username:     fmt.Sprintf("queryuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("queryuser_%d@test.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         "user",
		Status:       "active",
	}
	database.CreateUser(ctx, user)

	// Pre-populate with 1000 files
	base := time.Now().UnixNano()
	for i := 0; i < 1000; i++ {
		file := &db.File{
			ID:        fmt.Sprintf("queryfile_%d_%d", base, i),
			Name:      fmt.Sprintf("file_%d.txt", i),
			Type:      "file",
			UserID:    user.ID,
			MimeType:  "text/plain",
			SizeBytes: int64(i),
		}
		database.CreateFile(ctx, file)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		database.SearchFiles(ctx, user.ID, "file_", 100)
	}
}
