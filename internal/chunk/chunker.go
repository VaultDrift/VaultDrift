// Package chunk provides Content-Defined Chunking (CDC) using Rabin fingerprinting.
package chunk

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Chunker implements Content-Defined Chunking using Rabin fingerprints.
type Chunker struct {
	minSize   int
	avgSize   int
	maxSize   int
	window    int
	mask      uint64
	polynomial uint64
}

// ChunkInfo holds information about a chunk.
type ChunkInfo struct {
	Hash   string // SHA-256 hex
	Offset int64
	Size   int
}

// DefaultChunker creates a chunker with default parameters:
// min: 256KB, avg: 1MB, max: 4MB
func DefaultChunker() *Chunker {
	return NewChunker(256*1024, 1024*1024, 4*1024*1024)
}

// NewChunker creates a new chunker with specified parameters.
func NewChunker(min, avg, max int) *Chunker {
	// Calculate mask for average chunk size
	// mask = 2^bits - 1 where 2^bits = avg/min
	bits := 0
	size := avg / min
	for size > 1 {
		size >>= 1
		bits++
	}
	mask := (uint64(1) << bits) - 1

	return &Chunker{
		minSize:    min,
		avgSize:    avg,
		maxSize:    max,
		window:     48, // 48-byte window
		mask:       mask,
		polynomial: 0x3DA3358B4DC173, // Default Rabin polynomial
	}
}

// Chunk splits a reader into content-defined chunks.
// Returns a slice of ChunkInfo in order.
func (c *Chunker) Chunk(r io.Reader) ([]ChunkInfo, error) {
	br := bufio.NewReader(r)
	var chunks []ChunkInfo
	var offset int64

	for {
		chunk, err := c.readChunk(br, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if chunk == nil {
			break
		}

		chunks = append(chunks, *chunk)
		offset += int64(chunk.Size)
	}

	return chunks, nil
}

// readChunk reads a single chunk from the buffer.
func (c *Chunker) readChunk(br *bufio.Reader, offset int64) (*ChunkInfo, error) {
	data := make([]byte, 0, c.maxSize)
	window := make([]byte, c.window)
	var fingerprint uint64
	var pos int

	// Read bytes and look for chunk boundary
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			// End of input
			if len(data) == 0 {
				return nil, io.EOF
			}
			return c.createChunk(data, offset)
		}
		if err != nil {
			return nil, err
		}

		data = append(data, b)
		pos++

		// Update Rabin fingerprint
		if pos > c.window {
			// Remove oldest byte from window
			oldByte := window[0]
			copy(window, window[1:])
			window[c.window-1] = b
			fingerprint = c.updateFingerprint(fingerprint, oldByte, b)
		} else {
			window[pos-1] = b
			fingerprint = c.updateFingerprint(fingerprint, 0, b)
		}

		// Check for chunk boundary
		if len(data) >= c.minSize {
			// Check if fingerprint matches mask (boundary found)
			if (fingerprint & c.mask) == 0 || len(data) >= c.maxSize {
				return c.createChunk(data, offset)
			}
		}
	}
}

// createChunk creates a ChunkInfo from raw data.
func (c *Chunker) createChunk(data []byte, offset int64) (*ChunkInfo, error) {
	hash := sha256.Sum256(data)
	return &ChunkInfo{
		Hash:   hex.EncodeToString(hash[:]),
		Offset: offset,
		Size:   len(data),
	}, nil
}

// updateFingerprint updates the Rabin fingerprint with a new byte.
func (c *Chunker) updateFingerprint(fp uint64, outByte, inByte byte) uint64 {
	// Simple Rabin fingerprint update
	// fp = (fp << 1) + inByte - (outByte * highBit)
	// This is a simplified version; full implementation uses polynomial division

	// Shift and add new byte
	fp = (fp << 8) | uint64(inByte)

	// XOR with polynomial for mixing
	if outByte != 0 {
		fp ^= c.polynomial * uint64(outByte)
	}

	return fp
}

// ChunkWithData chunks a reader and returns chunk info and data.
func (c *Chunker) ChunkWithData(r io.Reader) ([]ChunkInfo, [][]byte, error) {
	br := bufio.NewReader(r)
	var chunks []ChunkInfo
	var data [][]byte
	var offset int64

	for {
		chunkBuf := make([]byte, 0, c.maxSize)
		window := make([]byte, c.window)
		var fingerprint uint64
		var pos int

		// Read until chunk boundary
		for {
			b, err := br.ReadByte()
			if err == io.EOF {
				if len(chunkBuf) == 0 {
					return chunks, data, nil
				}
				// Create final chunk
				hash := sha256.Sum256(chunkBuf)
				chunks = append(chunks, ChunkInfo{
					Hash:   hex.EncodeToString(hash[:]),
					Offset: offset,
					Size:   len(chunkBuf),
				})
				data = append(data, chunkBuf)
				return chunks, data, nil
			}
			if err != nil {
				return nil, nil, err
			}

			chunkBuf = append(chunkBuf, b)
			pos++

			// Update fingerprint
			if pos > c.window {
				oldByte := window[0]
				copy(window, window[1:])
				window[c.window-1] = b
				fingerprint = c.updateFingerprint(fingerprint, oldByte, b)
			} else {
				window[pos-1] = b
				fingerprint = c.updateFingerprint(fingerprint, 0, b)
			}

			// Check boundary
			if len(chunkBuf) >= c.minSize {
				if (fingerprint & c.mask) == 0 || len(chunkBuf) >= c.maxSize {
					break
				}
			}
		}

		// Create chunk
		hash := sha256.Sum256(chunkBuf)
		chunks = append(chunks, ChunkInfo{
			Hash:   hex.EncodeToString(hash[:]),
			Offset: offset,
			Size:   len(chunkBuf),
		})
		data = append(data, chunkBuf)
		offset += int64(len(chunkBuf))
	}
}
