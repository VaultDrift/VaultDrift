package chunk

import (
	"context"
	"fmt"
)

// Deduplicator handles chunk deduplication.
type Deduplicator struct {
	chunkIndex ChunkIndex
}

// ChunkIndex defines the interface for checking chunk existence.
type ChunkIndex interface {
	ChunkExists(ctx context.Context, hash string) (bool, error)
	CreateChunk(ctx context.Context, chunk *ChunkInfo, backend string) error
	IncrementRefCount(ctx context.Context, hash string) error
}

// NewDeduplicator creates a new deduplicator.
func NewDeduplicator(index ChunkIndex) *Deduplicator {
	return &Deduplicator{chunkIndex: index}
}

// DeduplicateResult holds the result of deduplication.
type DeduplicateResult struct {
	NewChunks     []ChunkInfo // Chunks that need to be uploaded
	ExistingHashes []string    // Hashes that already exist
	AllHashes     []string    // All chunk hashes in order
}

// Process checks which chunks are new and which already exist.
func (d *Deduplicator) Process(ctx context.Context, chunks []ChunkInfo) (*DeduplicateResult, error) {
	result := &DeduplicateResult{
		NewChunks:      make([]ChunkInfo, 0),
		ExistingHashes: make([]string, 0),
		AllHashes:      make([]string, len(chunks)),
	}

	for i, chunk := range chunks {
		result.AllHashes[i] = chunk.Hash

		exists, err := d.chunkIndex.ChunkExists(ctx, chunk.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to check chunk existence: %w", err)
		}

		if exists {
			// Chunk already exists, increment reference count
			if err := d.chunkIndex.IncrementRefCount(ctx, chunk.Hash); err != nil {
				return nil, fmt.Errorf("failed to increment ref count: %w", err)
			}
			result.ExistingHashes = append(result.ExistingHashes, chunk.Hash)
		} else {
			// New chunk needs to be uploaded
			result.NewChunks = append(result.NewChunks, chunk)
		}
	}

	return result, nil
}

// DeduplicateStats holds statistics about deduplication.
type DeduplicateStats struct {
	TotalChunks    int
	NewChunks      int
	ExistingChunks int
	DedupRatio     float64
}

// CalculateStats calculates deduplication statistics.
func CalculateStats(totalChunks, newChunks int) DeduplicateStats {
	existingChunks := totalChunks - newChunks
	ratio := 0.0
	if totalChunks > 0 {
		ratio = float64(existingChunks) / float64(totalChunks) * 100
	}

	return DeduplicateStats{
		TotalChunks:    totalChunks,
		NewChunks:      newChunks,
		ExistingChunks: existingChunks,
		DedupRatio:     ratio,
	}
}
