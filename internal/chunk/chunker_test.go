package chunk

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

func TestChunker(t *testing.T) {
	tests := []struct {
		name     string
		min      int
		avg      int
		max      int
		dataSize int
	}{
		{
			name:     "small file",
			min:      256,
			avg:      1024,
			max:      4096,
			dataSize: 10000,
		},
		{
			name:     "default params",
			min:      256 * 1024,
			avg:      1024 * 1024,
			max:      4 * 1024 * 1024,
			dataSize: 10 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataSize)
			if _, err := rand.Read(data); err != nil {
				t.Fatalf("failed to generate random data: %v", err)
			}

			chunker := NewChunker(tt.min, tt.avg, tt.max)
			chunks, err := chunker.Chunk(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("Chunk() error = %v", err)
			}

			if len(chunks) == 0 {
				t.Error("expected at least one chunk, got none")
			}

			var totalSize int64
			for i, c := range chunks {
				if c.Size < 0 {
					t.Errorf("chunk %d: size %d is negative", i, c.Size)
				}
				if c.Size > tt.max {
					t.Errorf("chunk %d: size %d exceeds max %d", i, c.Size, tt.max)
				}
				if c.Hash == "" {
					t.Errorf("chunk %d: hash is empty", i)
				}
				totalSize += int64(c.Size)
			}

			if totalSize != int64(tt.dataSize) {
				t.Errorf("total chunk size %d != input size %d", totalSize, tt.dataSize)
			}

			var expectedOffset int64
			for i, c := range chunks {
				if c.Offset != expectedOffset {
					t.Errorf("chunk %d: offset %d != expected %d", i, c.Offset, expectedOffset)
				}
				expectedOffset += int64(c.Size)
			}
		})
	}
}

func TestChunkerChunkWithData(t *testing.T) {
	data := make([]byte, 5*1024*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	chunker := NewChunker(256*1024, 1024*1024, 4*1024*1024)
	infos, chunks, err := chunker.ChunkWithData(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ChunkWithData() error = %v", err)
	}

	if len(infos) != len(chunks) {
		t.Errorf("infos length %d != chunks length %d", len(infos), len(chunks))
	}

	var reconstructed []byte
	for _, c := range chunks {
		reconstructed = append(reconstructed, c...)
	}

	if !bytes.Equal(data, reconstructed) {
		t.Error("reconstructed data doesn't match original")
	}
}

func TestRabinTableCache(t *testing.T) {
	poly := uint64(0x3DA3358B4DC173)

	table1 := getRabinTable(poly, 48)
	table2 := getRabinTable(poly, 48)

	if table1 != table2 {
		t.Error("expected same table instance from cache")
	}
}

func TestChunkBoundaryDeterminism(t *testing.T) {
	// Same data should produce same chunks
	data := make([]byte, 2*1024*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	chunker := NewChunker(64*1024, 256*1024, 1024*1024)

	chunks1, err := chunker.Chunk(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	chunks2, err := chunker.Chunk(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks1) != len(chunks2) {
		t.Errorf("different number of chunks: %d vs %d", len(chunks1), len(chunks2))
	}

	for i := range chunks1 {
		if chunks1[i].Hash != chunks2[i].Hash {
			t.Errorf("chunk %d: different hashes", i)
		}
		if chunks1[i].Size != chunks2[i].Size {
			t.Errorf("chunk %d: different sizes", i)
		}
	}
}

func TestDedupIndex(t *testing.T) {
	idx := NewDedupIndex()

	if idx.Exists("test-hash") {
		t.Error("hash should not exist initially")
	}

	idx.Add("test-hash")
	if !idx.Exists("test-hash") {
		t.Error("hash should exist after adding")
	}

	idx.Remove("test-hash")
	if idx.Exists("test-hash") {
		t.Error("hash should not exist after removing")
	}
}

// BenchmarkCDCOnly benchmarks only the Rabin fingerprinting boundary detection.
// Target: >500MB/s on single core.
func BenchmarkCDCOnly(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1 * 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			data := make([]byte, tc.size)
			if _, err := rand.Read(data); err != nil {
				b.Fatalf("failed to generate random data: %v", err)
			}

			chunker := NewChunker(256*1024, 1024*1024, 4*1024*1024)

			b.SetBytes(int64(tc.size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				cdcOnly(chunker, bytes.NewReader(data))
			}
		})
	}
}

// cdcOnly performs CDC boundary detection without hashing.
func cdcOnly(c *Chunker, r *bytes.Reader) {
	const bufSize = 4 * 1024 * 1024
	buf := make([]byte, bufSize)

	window := make([]byte, c.windowSize)
	var fingerprint uint64
	var windowPos int
	var windowFilled bool

	var chunkLen int

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				chunkLen++

				if windowFilled || windowPos >= c.windowSize {
					oldByte := window[0]
					copy(window, window[1:])
					window[c.windowSize-1] = b
					fingerprint = c.slideFingerprint(fingerprint, oldByte, b)
				} else {
					window[windowPos] = b
					windowPos++
					fingerprint = c.pushByte(fingerprint, b)
					if windowPos == c.windowSize {
						windowFilled = true
					}
				}

				if chunkLen >= c.minSize {
					if (fingerprint&c.mask) == 0 || chunkLen >= c.maxSize {
						chunkLen = 0
					}
				}
			}
		}

		if err != nil {
			break
		}
	}
}

// BenchmarkChunker benchmarks full chunking with SHA-256 hashing.
func BenchmarkChunker(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1 * 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			data := make([]byte, tc.size)
			if _, err := rand.Read(data); err != nil {
				b.Fatalf("failed to generate random data: %v", err)
			}

			chunker := NewChunker(256*1024, 1024*1024, 4*1024*1024)

			b.SetBytes(int64(tc.size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := chunker.Chunk(bytes.NewReader(data))
				if err != nil {
					b.Fatalf("Chunk() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkChunkerParallel benchmarks parallel chunking throughput.
func BenchmarkChunkerParallel(b *testing.B) {
	data := make([]byte, 100*1024*1024)
	if _, err := rand.Read(data); err != nil {
		b.Fatalf("failed to generate random data: %v", err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		chunker := NewChunker(256*1024, 1024*1024, 4*1024*1024)
		for pb.Next() {
			_, err := chunker.Chunk(bytes.NewReader(data))
			if err != nil {
				b.Fatalf("Chunk() error = %v", err)
			}
		}
	})
}

// BenchmarkSHA256 benchmarks SHA-256 hashing for comparison.
func BenchmarkSHA256(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1 * 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			data := make([]byte, tc.size)
			if _, err := rand.Read(data); err != nil {
				b.Fatalf("failed to generate random data: %v", err)
			}

			b.SetBytes(int64(tc.size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				sha256.Sum256(data)
			}
		})
	}
}
