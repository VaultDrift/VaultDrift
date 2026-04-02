package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/chunk"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/crypto"
	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// BenchmarkChunking benchmarks the CDC chunking algorithm
func BenchmarkChunking(b *testing.B) {
	chunker := chunk.DefaultChunker()

	// Generate test data of different sizes
	sizes := []int{
		1024 * 1024,       // 1MB
		10 * 1024 * 1024,  // 10MB
		100 * 1024 * 1024, // 100MB
	}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(fmt.Sprintf("Chunk_%dMB", size/1024/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, err := chunker.ChunkWithData(bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkStoragePut benchmarks storage backend write performance
func BenchmarkStoragePut(b *testing.B) {
	tmpDir := b.TempDir()

	store, err := storage.NewBackend(config.StorageConfig{
		Backend: "local",
		Local:   config.LocalConfig{DataDir: tmpDir + "/storage"},
	})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	sizes := []int{1024, 64 * 1024, 1024 * 1024} // 1KB, 64KB, 1MB

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)
		hash := fmt.Sprintf("bench_%d_%d", size, time.Now().UnixNano())

		b.Run(fmt.Sprintf("Put_%dKB", size/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("%s_%d", hash, i)
				if err := store.Put(ctx, key, data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkStorageGet benchmarks storage backend read performance
func BenchmarkStorageGet(b *testing.B) {
	tmpDir := b.TempDir()

	store, err := storage.NewBackend(config.StorageConfig{
		Backend: "local",
		Local:   config.LocalConfig{DataDir: tmpDir + "/storage"},
	})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()
	sizes := []int{1024, 64 * 1024, 1024 * 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)
		hash := fmt.Sprintf("bench_get_%d", size)
		store.Put(ctx, hash, data)

		b.Run(fmt.Sprintf("Get_%dKB", size/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := store.Get(ctx, hash)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEncryption benchmarks encryption performance
func BenchmarkEncryption(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)

	sizes := []int{1024, 64 * 1024, 1024 * 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(fmt.Sprintf("Encrypt_%dKB", size/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := crypto.Encrypt(data, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDecryption benchmarks decryption performance
func BenchmarkDecryption(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)

	sizes := []int{1024, 64 * 1024, 1024 * 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)
		encrypted, _ := crypto.Encrypt(data, key)

		b.Run(fmt.Sprintf("Decrypt_%dKB", size/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := crypto.Decrypt(encrypted, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDatabaseOperations benchmarks database performance
func BenchmarkDatabaseOperations(b *testing.B) {
	tmpDir := b.TempDir()
	database, err := db.Open(db.Config{Path: tmpDir + "/bench.db"})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// Benchmark user creation
	b.Run("CreateUser", func(b *testing.B) {
		base := time.Now().UnixNano()
		for i := 0; i < b.N; i++ {
			user := &db.User{
				Username:     fmt.Sprintf("user_%d_%d", base, i),
				Email:        fmt.Sprintf("user_%d_%d@test.com", base, i),
				PasswordHash: "hash",
				Role:         "user",
				Status:       "active",
			}
			if err := database.CreateUser(ctx, user); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark file creation
	b.Run("CreateFile", func(b *testing.B) {
		user := &db.User{Username: fmt.Sprintf("fileowner_%d", time.Now().UnixNano()), Email: fmt.Sprintf("owner_%d@test.com", time.Now().UnixNano()), PasswordHash: "hash", Role: "user", Status: "active"}
		database.CreateUser(ctx, user)
		base := time.Now().UnixNano()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file := &db.File{
				ID:        fmt.Sprintf("file_%d_%d", base, i),
				Name:      fmt.Sprintf("file_%d_%d.txt", base, i),
				Type:      "file",
				SizeBytes: 1024,
				MimeType:  "text/plain",
				UserID:    user.ID,
			}
			if err := database.CreateFile(ctx, file); err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark chunk creation
	b.Run("CreateChunk", func(b *testing.B) {
		base := time.Now().UnixNano()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			chunk := &db.Chunk{
				Hash:           fmt.Sprintf("hash_%d_%d", base, i),
				SizeBytes:      1024,
				StorageBackend: "local",
				StoragePath:    fmt.Sprintf("chunks/ha/hash_%d_%d", base, i),
				RefCount:       1,
			}
			if err := database.CreateChunk(ctx, chunk); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkVFSOperations benchmarks VFS operations
func BenchmarkVFSOperations(b *testing.B) {
	tmpDir := b.TempDir()
	database, err := db.Open(db.Config{Path: tmpDir + "/vfsbench.db"})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	vfsService := vfs.NewVFS(database)
	ctx := context.Background()

	user := &db.User{Username: fmt.Sprintf("vfsuser_%d", time.Now().UnixNano()), Email: fmt.Sprintf("vfs_%d@test.com", time.Now().UnixNano()), PasswordHash: "hash", Role: "user", Status: "active"}
	database.CreateUser(ctx, user)

	// Benchmark folder creation
	b.Run("CreateFolder", func(b *testing.B) {
		base := time.Now().UnixNano()
		for i := 0; i < b.N; i++ {
			_, err := vfsService.CreateFolder(ctx, user.ID, "", fmt.Sprintf("folder_%d_%d", base, i))
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark file creation
	b.Run("CreateFile", func(b *testing.B) {
		base := time.Now().UnixNano()
		for i := 0; i < b.N; i++ {
			_, err := vfsService.CreateFile(ctx, user.ID, "", fmt.Sprintf("file_%d_%d.txt", base, i), "text/plain", 1024)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Benchmark listing
	folder, _ := vfsService.CreateFolder(ctx, user.ID, "", "listtest")
	for i := 0; i < 100; i++ {
		vfsService.CreateFile(ctx, user.ID, folder.ID, fmt.Sprintf("file_%d.txt", i), "text/plain", 1024)
	}

	b.Run("ListDirectory_100", func(b *testing.B) {
		opts := db.ListOpts{Limit: 100}
		for i := 0; i < b.N; i++ {
			_, err := vfsService.ListDirectory(ctx, user.ID, folder.ID, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDeduplication benchmarks deduplication effectiveness
func BenchmarkDeduplication(b *testing.B) {
	tmpDir := b.TempDir()
	database, err := db.Open(db.Config{Path: tmpDir + "/dedup.db"})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	store, err := storage.NewBackend(config.StorageConfig{
		Backend: "local",
		Local:   config.LocalConfig{DataDir: tmpDir + "/storage"},
	})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	chunker := chunk.DefaultChunker()
	ctx := context.Background()

	// Create identical files multiple times
	data := make([]byte, 10*1024*1024) // 10MB
	rand.Read(data)

	b.Run("StoreIdenticalFiles", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cdcChunks, cdcData, err := chunker.ChunkWithData(bytes.NewReader(data))
			if err != nil {
				b.Fatal(err)
			}

			// Store chunks with deduplication
			for j, cdcChunk := range cdcChunks {
				exists, _ := database.ChunkExists(ctx, cdcChunk.Hash)
				if exists {
					database.IncrementRefCount(ctx, cdcChunk.Hash)
				} else {
					store.Put(ctx, cdcChunk.Hash, cdcData[j])
					database.CreateChunk(ctx, &db.Chunk{
						Hash:           cdcChunk.Hash,
						SizeBytes:      int64(cdcChunk.Size),
						StorageBackend: "local",
						RefCount:       1,
					})
				}
			}
		}
	})

	// Check actual storage used vs logical size
	stats, _ := store.Stats(ctx)
	b.Logf("Storage efficiency: %d bytes stored for %d bytes logical (%d files)",
		stats.UsedBytes, int64(b.N)*10*1024*1024, b.N)
}

// BenchmarkEndToEndUpload benchmarks full upload pipeline
func BenchmarkEndToEndUpload(b *testing.B) {
	tmpDir := b.TempDir()
	database, err := db.Open(db.Config{Path: tmpDir + "/e2e.db"})
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	store, err := storage.NewBackend(config.StorageConfig{
		Backend: "local",
		Local:   config.LocalConfig{DataDir: tmpDir + "/storage"},
	})
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	vfsService := vfs.NewVFS(database)
	chunker := chunk.DefaultChunker()
	ctx := context.Background()

	user := &db.User{Username: fmt.Sprintf("e2euser_%d", time.Now().UnixNano()), Email: fmt.Sprintf("e2e_%d@test.com", time.Now().UnixNano()), PasswordHash: "hash", Role: "user", Status: "active"}
	database.CreateUser(ctx, user)

	sizes := []int{1024 * 1024, 10 * 1024 * 1024, 50 * 1024 * 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(fmt.Sprintf("Upload_%dMB", size/1024/1024), func(b *testing.B) {
			base := time.Now().UnixNano()
			for i := 0; i < b.N; i++ {
				// Create file
				file, err := vfsService.CreateFile(ctx, user.ID, "", fmt.Sprintf("upload_%d_%d.bin", base, i), "application/octet-stream", int64(size))
				if err != nil {
					b.Fatal(err)
				}

				// Chunk and store
				cdcChunks, cdcData, err := chunker.ChunkWithData(bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}

				chunkHashes := make([]string, len(cdcChunks))
				for j, cdcChunk := range cdcChunks {
					chunkHashes[j] = cdcChunk.Hash
					store.Put(ctx, cdcChunk.Hash, cdcData[j])
					database.CreateChunk(ctx, &db.Chunk{
						Hash:           cdcChunk.Hash,
						SizeBytes:      int64(cdcChunk.Size),
						StorageBackend: "local",
						RefCount:       1,
					})
				}

				// Create manifest
				manifest := &db.Manifest{
					ID:         fmt.Sprintf("manifest_%d_%d", base, i),
					FileID:     file.ID,
					Version:    1,
					SizeBytes:  int64(size),
					Chunks:     chunkHashes,
					ChunkCount: len(chunkHashes),
				}
				database.CreateManifest(ctx, manifest)
				database.UpdateFile(ctx, file.ID, map[string]any{"manifest_id": manifest.ID})
			}
		})
	}
}
