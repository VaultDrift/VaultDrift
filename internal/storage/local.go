package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vaultdrift/vaultdrift/internal/config"
)

// LocalBackend implements Backend for local filesystem storage.
type LocalBackend struct {
	dataDir string
	mu      sync.RWMutex
}

// NewLocalBackend creates a new local filesystem backend.
func NewLocalBackend(cfg config.LocalConfig) (*LocalBackend, error) {
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &LocalBackend{
		dataDir: cfg.DataDir,
	}, nil
}

// keyToPath converts a hash key to a filesystem path.
// Format: {dataDir}/chunks/{hash[:2]}/{hash[2:]}.chunk
func (b *LocalBackend) keyToPath(key string) string {
	if len(key) < 2 {
		return filepath.Join(b.dataDir, "chunks", key+".chunk")
	}
	return filepath.Join(b.dataDir, "chunks", key[:2], key[2:]+".chunk")
}

// Put stores a chunk using atomic write (write to temp, then rename).
func (b *LocalBackend) Put(ctx context.Context, key string, data []byte) error {
	path := b.keyToPath(key)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath) // Cleanup on error
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// Get retrieves a chunk by key.
func (b *LocalBackend) Get(ctx context.Context, key string) ([]byte, error) {
	path := b.keyToPath(key)

	data, err := os.ReadFile(path) // #nosec G304 - path constructed from hash key
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chunk not found")
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete removes a chunk.
func (b *LocalBackend) Delete(ctx context.Context, key string) error {
	path := b.keyToPath(key)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("chunk not found")
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Try to remove empty parent directory
	dir := filepath.Dir(path)
	_ = os.Remove(dir) // Ignore error - directory might not be empty

	return nil
}

// Exists checks if a chunk exists.
func (b *LocalBackend) Exists(ctx context.Context, key string) (bool, error) {
	path := b.keyToPath(key)

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check file: %w", err)
}

// List returns chunk keys with given prefix.
func (b *LocalBackend) List(ctx context.Context, prefix string) ([]string, error) {
	chunksDir := filepath.Join(b.dataDir, "chunks")

	var keys []string
	err := filepath.Walk(chunksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".chunk") {
			return nil
		}

		// Extract key from path
		rel, _ := filepath.Rel(chunksDir, path)
		// rel format: "ab/cdef...chunk"
		key := strings.ReplaceAll(rel, string(filepath.Separator), "")
		key = strings.TrimSuffix(key, ".chunk")

		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return keys, nil
}

// Stats returns storage statistics.
func (b *LocalBackend) Stats(ctx context.Context) (*StorageStats, error) {
	chunksDir := filepath.Join(b.dataDir, "chunks")

	var totalBytes int64
	var chunkCount int64

	err := filepath.Walk(chunksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".chunk") {
			totalBytes += info.Size()
			chunkCount++
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Get filesystem stats for total capacity
	// This is platform-specific; for now return 0 for total
	return &StorageStats{
		TotalBytes:  0, // Would need platform-specific code
		UsedBytes:   totalBytes,
		ChunkCount:  chunkCount,
		BackendType: "local",
	}, nil
}
