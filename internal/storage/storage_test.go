package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/vaultdrift/vaultdrift/internal/config"
)

func TestLocalBackend(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	cfg := config.LocalConfig{
		DataDir: tmpDir,
	}

	backend, err := NewLocalBackend(cfg)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	t.Run("PutAndGet", func(t *testing.T) {
		key := "test-chunk-1"
		data := []byte("hello world test data")

		// Put
		if err := backend.Put(ctx, key, data); err != nil {
			t.Errorf("Put failed: %v", err)
		}

		// Get
		got, err := backend.Get(ctx, key)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}

		if string(got) != string(data) {
			t.Errorf("Get returned wrong data: got %s, want %s", got, data)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		key := "test-exists"
		data := []byte("exists test")

		// Before put
		exists, err := backend.Exists(ctx, key)
		if err != nil {
			t.Errorf("Exists check failed: %v", err)
		}
		if exists {
			t.Error("Exists should return false for non-existent key")
		}

		// Put
		if err := backend.Put(ctx, key, data); err != nil {
			t.Errorf("Put failed: %v", err)
		}

		// After put
		exists, err = backend.Exists(ctx, key)
		if err != nil {
			t.Errorf("Exists check failed: %v", err)
		}
		if !exists {
			t.Error("Exists should return true for existing key")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "test-delete"
		data := []byte("delete test")

		// Put
		if err := backend.Put(ctx, key, data); err != nil {
			t.Errorf("Put failed: %v", err)
		}

		// Delete
		if err := backend.Delete(ctx, key); err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify deletion
		_, err := backend.Get(ctx, key)
		if err == nil {
			t.Error("Get should fail after delete")
		}
	})

	t.Run("List", func(t *testing.T) {
		// Put multiple chunks
		for i := 0; i < 5; i++ {
			key := "prefix-chunk-" + string(rune('a'+i))
			if err := backend.Put(ctx, key, []byte("data")); err != nil {
				t.Errorf("Put failed: %v", err)
			}
		}

		// List with prefix
		keys, err := backend.List(ctx, "prefix")
		if err != nil {
			t.Errorf("List failed: %v", err)
		}

		if len(keys) != 5 {
			t.Errorf("List returned wrong number of keys: got %d, want 5", len(keys))
		}
	})

	t.Run("Stats", func(t *testing.T) {
		stats, err := backend.Stats(ctx)
		if err != nil {
			t.Errorf("Stats failed: %v", err)
		}

		if stats.BackendType != "local" {
			t.Errorf("Wrong backend type: got %s, want local", stats.BackendType)
		}
	})
}

func TestBackendPathGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.LocalConfig{DataDir: tmpDir}

	backend, err := NewLocalBackend(cfg)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	tests := []struct {
		key     string
		wantDir string
	}{
		{
			key:     "abc123def456",
			wantDir: "ab",
		},
		{
			key:     "short",
			wantDir: "sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotPath := backend.keyToPath(tt.key)

			// Check that path contains the hash prefix directory
			if !strings.Contains(gotPath, tt.wantDir) {
				t.Errorf("keyToPath(%s) = %s, should contain %s", tt.key, gotPath, tt.wantDir)
			}

			// Check that path ends with .chunk
			if !strings.HasSuffix(gotPath, ".chunk") {
				t.Errorf("keyToPath(%s) = %s, should end with .chunk", tt.key, gotPath)
			}
		})
	}
}

func TestBackendConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.LocalConfig{DataDir: tmpDir}

	backend, err := NewLocalBackend(cfg)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			key := "concurrent-" + string(rune('0'+i))
			data := []byte("data-" + string(rune('0'+i)))
			if err := backend.Put(ctx, key, data); err != nil {
				t.Errorf("Concurrent Put failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all writes
	for i := 0; i < 10; i++ {
		key := "concurrent-" + string(rune('0'+i))
		_, err := backend.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get %s: %v", key, err)
		}
	}
}
