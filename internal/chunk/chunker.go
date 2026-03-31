package chunk

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
)

// rabinTable holds precomputed tables for Rabin fingerprinting.
type rabinTable struct {
	polynomial uint64
	degree     uint
	outTable   [256]uint64
	modTable   [256]uint64
}

// rabinTables caches computed tables for common polynomials.
var rabinTables = struct {
	sync.RWMutex
	tables map[uint64]*rabinTable
}{tables: make(map[uint64]*rabinTable)}

func getRabinTable(poly uint64, windowSize int) *rabinTable {
	rabinTables.RLock()
	table, ok := rabinTables.tables[poly]
	rabinTables.RUnlock()
	if ok {
		return table
	}

	rabinTables.Lock()
	defer rabinTables.Unlock()

	if table, ok = rabinTables.tables[poly]; ok {
		return table
	}

	table = createRabinTable(poly, windowSize)
	rabinTables.tables[poly] = table
	return table
}

func createRabinTable(poly uint64, windowSize int) *rabinTable {
	table := &rabinTable{polynomial: poly}

	degree := uint(63)
	for degree > 0 && ((poly>>degree)&1) == 0 {
		degree--
	}
	table.degree = degree
	pDegree := uint64(1) << degree

	var outPoly uint64 = 1
	for i := 0; i < windowSize; i++ {
		outPoly <<= 1
		if (outPoly & pDegree) != 0 {
			outPoly ^= poly
		}
	}

	for b := range 256 {
		var result uint64
		bb := uint64(b)
		for i := range 8 {
			if (bb>>i)&1 != 0 {
				result ^= outPoly << i
			}
		}
		for i := uint(63); i >= degree; i-- {
			if (result>>i)&1 != 0 {
				result ^= poly << (i - degree)
			}
		}
		table.outTable[b] = result
	}

	for b := range 256 {
		var result uint64
		bb := uint64(b)
		for i := range 8 {
			if (bb>>i)&1 != 0 {
				result ^= poly << i
			}
		}
		for i := uint(63); i >= degree; i-- {
			if (result>>i)&1 != 0 {
				result ^= poly << (i - degree)
			}
		}
		table.modTable[b] = result
	}

	return table
}

// Chunker implements Content-Defined Chunking using Rabin fingerprints.
type Chunker struct {
	minSize    int
	avgSize    int
	maxSize    int
	windowSize int
	mask       uint64
	polynomial uint64
	table      *rabinTable
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
	bits := 0
	size := avg
	for size > 1 {
		size >>= 1
		bits++
	}
	bits = bits - 2
	if bits < 8 {
		bits = 8
	}
	mask := (uint64(1) << bits) - 1

	polynomial := uint64(0x3DA3358B4DC173)
	windowSize := 48
	table := getRabinTable(polynomial, windowSize)

	return &Chunker{
		minSize:    min,
		avgSize:    avg,
		maxSize:    max,
		windowSize: windowSize,
		mask:       mask,
		polynomial: polynomial,
		table:      table,
	}
}

// Chunk splits a reader into content-defined chunks.
func (c *Chunker) Chunk(r io.Reader) ([]ChunkInfo, error) {
	var chunks []ChunkInfo
	var offset int64

	const bufSize = 4 * 1024 * 1024
	buf := make([]byte, bufSize)

	window := make([]byte, c.windowSize)
	var fingerprint uint64
	var windowPos int
	var windowFilled bool

	var chunkBuf []byte
	chunkBuf = make([]byte, 0, c.maxSize)
	chunkOffset := offset

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				chunkBuf = append(chunkBuf, b)
				offset++

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

				if len(chunkBuf) >= c.minSize {
					if (fingerprint&c.mask) == 0 || len(chunkBuf) >= c.maxSize {
						hash := sha256.Sum256(chunkBuf)
						chunks = append(chunks, ChunkInfo{
							Hash:   hex.EncodeToString(hash[:]),
							Offset: chunkOffset,
							Size:   len(chunkBuf),
						})

						chunkBuf = chunkBuf[:0]
						chunkOffset = offset
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if len(chunkBuf) > 0 {
		hash := sha256.Sum256(chunkBuf)
		chunks = append(chunks, ChunkInfo{
			Hash:   hex.EncodeToString(hash[:]),
			Offset: chunkOffset,
			Size:   len(chunkBuf),
		})
	}

	return chunks, nil
}

// pushByte adds a byte to the fingerprint.
func (c *Chunker) pushByte(fp uint64, b byte) uint64 {
	fp <<= 1
	if (fp & (uint64(1) << c.table.degree)) != 0 {
		fp ^= c.polynomial
	}
	return fp ^ uint64(b)
}

// slideFingerprint updates fingerprint by removing oldByte and adding newByte.
func (c *Chunker) slideFingerprint(fp uint64, oldByte, newByte byte) uint64 {
	fp ^= c.table.outTable[oldByte]
	fp <<= 1
	if (fp & (uint64(1) << c.table.degree)) != 0 {
		fp ^= c.polynomial
	}
	return fp ^ uint64(newByte)
}

// ChunkWithData chunks a reader and returns chunk info and data.
func (c *Chunker) ChunkWithData(r io.Reader) ([]ChunkInfo, [][]byte, error) {
	var chunks []ChunkInfo
	var data [][]byte
	var offset int64

	const bufSize = 4 * 1024 * 1024
	buf := make([]byte, bufSize)

	window := make([]byte, c.windowSize)
	var fingerprint uint64
	var windowPos int
	var windowFilled bool

	var chunkBuf []byte
	chunkBuf = make([]byte, 0, c.maxSize)
	chunkOffset := offset

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				chunkBuf = append(chunkBuf, b)
				offset++

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

				if len(chunkBuf) >= c.minSize {
					if (fingerprint&c.mask) == 0 || len(chunkBuf) >= c.maxSize {
						chunkCopy := make([]byte, len(chunkBuf))
						copy(chunkCopy, chunkBuf)

						hash := sha256.Sum256(chunkCopy)
						chunks = append(chunks, ChunkInfo{
							Hash:   hex.EncodeToString(hash[:]),
							Offset: chunkOffset,
							Size:   len(chunkCopy),
						})
						data = append(data, chunkCopy)

						chunkBuf = chunkBuf[:0]
						chunkOffset = offset
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}

	if len(chunkBuf) > 0 {
		chunkCopy := make([]byte, len(chunkBuf))
		copy(chunkCopy, chunkBuf)

		hash := sha256.Sum256(chunkCopy)
		chunks = append(chunks, ChunkInfo{
			Hash:   hex.EncodeToString(hash[:]),
			Offset: chunkOffset,
			Size:   len(chunkCopy),
		})
		data = append(data, chunkCopy)
	}

	return chunks, data, nil
}

// ChunkStream processes data through a writer interface and reports chunks via callback.
// This allows integration with hash.Hash for streaming hash computation.
func (c *Chunker) ChunkStream(r io.Reader, onChunk func(offset int64, size int, hash string, data []byte) error) error {
	const bufSize = 4 * 1024 * 1024
	buf := make([]byte, bufSize)

	window := make([]byte, c.windowSize)
	var fingerprint uint64
	var windowPos int
	var windowFilled bool

	var chunkBuf []byte
	chunkBuf = make([]byte, 0, c.maxSize)
	chunkOffset := int64(0)
	hasher := sha256.New()

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				chunkBuf = append(chunkBuf, b)
				hasher.Write([]byte{b})

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

				if len(chunkBuf) >= c.minSize {
					if (fingerprint&c.mask) == 0 || len(chunkBuf) >= c.maxSize {
						hash := hex.EncodeToString(hasher.Sum(nil))
						chunkCopy := make([]byte, len(chunkBuf))
						copy(chunkCopy, chunkBuf)

						if err := onChunk(chunkOffset, len(chunkBuf), hash, chunkCopy); err != nil {
							return err
						}

						hasher.Reset()
						chunkBuf = chunkBuf[:0]
						chunkOffset += int64(len(chunkBuf))
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if len(chunkBuf) > 0 {
		hash := hex.EncodeToString(hasher.Sum(nil))
		chunkCopy := make([]byte, len(chunkBuf))
		copy(chunkCopy, chunkBuf)
		if err := onChunk(chunkOffset, len(chunkBuf), hash, chunkCopy); err != nil {
			return err
		}
	}

	return nil
}

// DedupIndex provides fast chunk existence checking for deduplication.
type DedupIndex struct {
	mu     sync.RWMutex
	exists map[string]struct{}
}

// NewDedupIndex creates a new deduplication index.
func NewDedupIndex() *DedupIndex {
	return &DedupIndex{
		exists: make(map[string]struct{}),
	}
}

// Add adds a chunk hash to the index.
func (d *DedupIndex) Add(hash string) {
	d.mu.Lock()
	d.exists[hash] = struct{}{}
	d.mu.Unlock()
}

// Exists checks if a chunk hash exists in the index.
func (d *DedupIndex) Exists(hash string) bool {
	d.mu.RLock()
	_, ok := d.exists[hash]
	d.mu.RUnlock()
	return ok
}

// Remove removes a chunk hash from the index.
func (d *DedupIndex) Remove(hash string) {
	d.mu.Lock()
	delete(d.exists, hash)
	d.mu.Unlock()
}
