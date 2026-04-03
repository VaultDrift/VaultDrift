package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/circuit"
	"github.com/vaultdrift/vaultdrift/internal/config"
	"github.com/vaultdrift/vaultdrift/internal/storage/s3client"
)

const defaultTimeout = 30 * time.Second

// S3Backend implements Backend for S3-compatible storage.
type S3Backend struct {
	config  config.S3Config
	client  *s3client.Client
	breaker *circuit.Breaker
}

// NewS3Backend creates a new S3 storage backend.
func NewS3Backend(cfg config.S3Config) (*S3Backend, error) {
	clientCfg := s3client.Config{
		Endpoint:     cfg.Endpoint,
		Bucket:       cfg.Bucket,
		Region:       cfg.Region,
		AccessKey:    cfg.AccessKey,
		SecretKey:    cfg.SecretKey,
		UsePathStyle: cfg.UsePathStyle,
	}

	client := s3client.NewClient(clientCfg)

	// Create circuit breaker for S3 operations
	breaker := circuit.New("s3", circuit.Config{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		MaxRetries:  2,
		RetryDelay:  1 * time.Second,
	})

	// Verify bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if err := client.HeadBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to S3 bucket: %w", err)
	}

	return &S3Backend{
		config:  cfg,
		client:  client,
		breaker: breaker,
	}, nil
}

// Put stores a chunk with circuit breaker protection.
func (b *S3Backend) Put(ctx context.Context, key string, data []byte) error {
	objectKey := b.objectKey(key)

	return b.breaker.Execute(ctx, func() error {
		return b.client.PutObject(ctx, objectKey, strings.NewReader(string(data)), int64(len(data)))
	})
}

// Get retrieves a chunk with circuit breaker protection.
func (b *S3Backend) Get(ctx context.Context, key string) ([]byte, error) {
	objectKey := b.objectKey(key)

	return circuit.ExecuteWithResult(b.breaker, ctx, func() ([]byte, error) {
		reader, _, err := b.client.GetObject(ctx, objectKey)
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		return io.ReadAll(reader)
	})
}

// Delete removes a chunk with circuit breaker protection.
func (b *S3Backend) Delete(ctx context.Context, key string) error {
	objectKey := b.objectKey(key)

	return b.breaker.Execute(ctx, func() error {
		return b.client.DeleteObject(ctx, objectKey)
	})
}

// Exists checks if a chunk exists with circuit breaker protection.
func (b *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	objectKey := b.objectKey(key)

	return circuit.ExecuteWithResult(b.breaker, ctx, func() (bool, error) {
		_, err := b.client.HeadObject(ctx, objectKey)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// List returns chunk keys with given prefix.
func (b *S3Backend) List(ctx context.Context, prefix string) ([]string, error) {
	objects, err := b.client.ListObjectsV2(ctx, prefix)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(objects))
	for i, obj := range objects {
		// Strip the prefix path to get just the chunk key
		keys[i] = b.chunkKeyFromObject(obj.Key)
	}
	return keys, nil
}

// Type returns the storage backend type
func (b *S3Backend) Type() string {
	return "s3"
}

// Stats returns storage statistics.
func (b *S3Backend) Stats(ctx context.Context) (*StorageStats, error) {
	objects, err := b.client.ListObjectsV2(ctx, "")
	if err != nil {
		return nil, err
	}

	var totalSize int64
	var count int64

	for _, obj := range objects {
		totalSize += obj.Size
		count++
	}

	return &StorageStats{
		TotalBytes:  totalSize,
		UsedBytes:   totalSize,
		ChunkCount:  count,
		BackendType: "s3",
	}, nil
}

// objectKey converts a chunk key to an S3 object key.
func (b *S3Backend) objectKey(key string) string {
	return key
}

// chunkKeyFromObject extracts the chunk key from an S3 object key.
func (b *S3Backend) chunkKeyFromObject(objectKey string) string {
	return objectKey
}
