package chunk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/vaultdrift/vaultdrift/internal/storage"
)

// Reassembler handles reassembling chunks into files.
type Reassembler struct {
	storage storage.Backend
}

// NewReassembler creates a new reassembler.
func NewReassembler(store storage.Backend) *Reassembler {
	return &Reassembler{storage: store}
}

// Reassemble streams chunks from storage and writes them to the output writer.
func (r *Reassembler) Reassemble(ctx context.Context, manifest *Manifest, w io.Writer) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	for i, chunk := range manifest.Chunks {
		data, err := r.storage.Get(ctx, chunk.Hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %d (%s): %w", i, chunk.Hash, err)
		}

		if len(data) != chunk.Size {
			return fmt.Errorf("chunk %d size mismatch: expected %d, got %d", i, chunk.Size, len(data))
		}

		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	return nil
}

// ReassembleToBuffer reassembles chunks into a byte buffer.
func (r *Reassembler) ReassembleToBuffer(ctx context.Context, manifest *Manifest) ([]byte, error) {
	buf := make([]byte, 0, manifest.Size)
	var offset int

	for i, chunk := range manifest.Chunks {
		data, err := r.storage.Get(ctx, chunk.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunk %d (%s): %w", i, chunk.Hash, err)
		}

		// Ensure buffer is large enough
		if offset+len(data) > len(buf) {
			newBuf := make([]byte, offset+len(data))
			copy(newBuf, buf)
			buf = newBuf
		}

		copy(buf[offset:], data)
		offset += len(data)
	}

	return buf[:offset], nil
}

// ReassembleRange reassembles a byte range from chunks.
func (r *Reassembler) ReassembleRange(ctx context.Context, manifest *Manifest, start, end int64, w io.Writer) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	if start < 0 || end > manifest.Size || start >= end {
		return fmt.Errorf("invalid range: %d-%d", start, end)
	}

	var currentOffset int64
	bytesWritten := int64(0)
	targetBytes := end - start

	for _, chunk := range manifest.Chunks {
		chunkStart := currentOffset
		chunkEnd := currentOffset + int64(chunk.Size)

		// Check if this chunk intersects with the requested range
		if chunkEnd <= start || chunkStart >= end {
			currentOffset = chunkEnd
			continue
		}

		// Fetch chunk
		data, err := r.storage.Get(ctx, chunk.Hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", chunk.Hash, err)
		}

		// Calculate slice within chunk
		sliceStart := int64(0)
		sliceEnd := int64(len(data))

		if chunkStart < start {
			sliceStart = start - chunkStart
		}
		if chunkEnd > end {
			sliceEnd = end - chunkStart
		}

		slice := data[sliceStart:sliceEnd]
		if _, err := w.Write(slice); err != nil {
			return fmt.Errorf("failed to write chunk data: %w", err)
		}

		bytesWritten += int64(len(slice))
		if bytesWritten >= targetBytes {
			break
		}

		currentOffset = chunkEnd
	}

	return nil
}

// CalculateManifestChecksum calculates the checksum of a reassembled file.
// This is expensive as it requires fetching all chunks.
func (r *Reassembler) CalculateManifestChecksum(ctx context.Context, manifest *Manifest) (string, error) {
	h := sha256.New()

	for _, chunk := range manifest.Chunks {
		data, err := r.storage.Get(ctx, chunk.Hash)
		if err != nil {
			return "", fmt.Errorf("failed to get chunk %s: %w", chunk.Hash, err)
		}
		h.Write(data)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
