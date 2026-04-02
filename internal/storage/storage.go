// Package storage provides the Storage Abstraction Layer (SAL) for VaultDrift.
// It defines a common interface for different storage backends (local filesystem, S3, etc.).
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/vaultdrift/vaultdrift/internal/config"
)

// Backend is the storage abstraction interface.
// All implementations must be safe for concurrent use.
type Backend interface {
	// Put stores a chunk blob. Key is the content hash.
	Put(ctx context.Context, key string, data []byte) error

	// Get retrieves a chunk blob by key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes a chunk blob.
	Delete(ctx context.Context, key string) error

	// Exists checks if a chunk exists.
	Exists(ctx context.Context, key string) (bool, error)

	// List returns chunk keys with given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Stats returns storage usage statistics.
	Stats(ctx context.Context) (*StorageStats, error)

	// Type returns the storage backend type name
	Type() string
}

// StorageStats holds storage statistics.
type StorageStats struct {
	TotalBytes  int64
	UsedBytes   int64
	ChunkCount  int64
	BackendType string
}

// NewBackend creates a storage backend based on configuration.
func NewBackend(cfg config.StorageConfig) (Backend, error) {
	switch cfg.Backend {
	case "local":
		return NewLocalBackend(cfg.Local)
	case "s3":
		return NewS3Backend(cfg.S3)
	case "ipfs":
		return NewIPFSBackend(cfg.IPFS)
	default:
		return nil, fmt.Errorf("unknown storage backend: %s", cfg.Backend)
	}
}

// ReaderFunc is an adapter to allow the use of ordinary functions as io.Reader.
type ReaderFunc func(p []byte) (n int, err error)

func (f ReaderFunc) Read(p []byte) (n int, err error) {
	return f(p)
}

// NOPCloser wraps an io.Reader and provides a no-op Close method.
type NOPCloser struct {
	io.Reader
}

func (NOPCloser) Close() error { return nil }
