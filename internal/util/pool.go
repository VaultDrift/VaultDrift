package util

import (
	"sync"
)

// Common buffer sizes for the pool tiers.
const (
	SmallBufferSize  = 4 * 1024        // 4KB
	MediumBufferSize = 64 * 1024       // 64KB
	LargeBufferSize  = 1024 * 1024     // 1MB
	XLargeBufferSize = 4 * 1024 * 1024 // 4MB
)

// BufferPool provides a pool of reusable byte buffers.
// It uses multiple tiers based on size to minimize allocations.
type BufferPool struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool
	xlarge sync.Pool
}

// NewBufferPool creates a new BufferPool.
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small: sync.Pool{
			New: func() interface{} {
				b := make([]byte, SmallBufferSize)
				return &b
			},
		},
		medium: sync.Pool{
			New: func() interface{} {
				b := make([]byte, MediumBufferSize)
				return &b
			},
		},
		large: sync.Pool{
			New: func() interface{} {
				b := make([]byte, LargeBufferSize)
				return &b
			},
		},
		xlarge: sync.Pool{
			New: func() interface{} {
				b := make([]byte, XLargeBufferSize)
				return &b
			},
		},
	}
}

// Get retrieves a buffer of at least the requested size from the pool.
// The returned buffer may be larger than requested.
func (p *BufferPool) Get(size int) []byte {
	if size <= SmallBufferSize {
		buf := p.small.Get().(*[]byte)
		return (*buf)[:size]
	}
	if size <= MediumBufferSize {
		buf := p.medium.Get().(*[]byte)
		return (*buf)[:size]
	}
	if size <= LargeBufferSize {
		buf := p.large.Get().(*[]byte)
		return (*buf)[:size]
	}
	if size <= XLargeBufferSize {
		buf := p.xlarge.Get().(*[]byte)
		return (*buf)[:size]
	}
	// Too large for pool, allocate directly
	return make([]byte, size)
}

// Put returns a buffer to the pool for reuse.
func (p *BufferPool) Put(buf []byte) {
	cap := cap(buf)
	switch {
	case cap == SmallBufferSize:
		p.small.Put(&buf)
	case cap == MediumBufferSize:
		p.medium.Put(&buf)
	case cap == LargeBufferSize:
		p.large.Put(&buf)
	case cap == XLargeBufferSize:
		p.xlarge.Put(&buf)
		// else: let GC collect oversized buffers
	}
}

// DefaultPool is the global default buffer pool.
var DefaultPool = NewBufferPool()

// GetBuffer retrieves a buffer from the default pool.
func GetBuffer(size int) []byte {
	return DefaultPool.Get(size)
}

// PutBuffer returns a buffer to the default pool.
func PutBuffer(buf []byte) {
	DefaultPool.Put(buf)
}

// Buffer represents a pooled byte buffer with automatic return.
type Buffer struct {
	data []byte
	pool *BufferPool
}

// NewBuffer creates a new pooled buffer.
func (p *BufferPool) NewBuffer(size int) *Buffer {
	return &Buffer{
		data: p.Get(size),
		pool: p,
	}
}

// Bytes returns the underlying byte slice.
func (b *Buffer) Bytes() []byte {
	return b.data
}

// Len returns the length of the buffer.
func (b *Buffer) Len() int {
	return len(b.data)
}

// Cap returns the capacity of the buffer.
func (b *Buffer) Cap() int {
	return cap(b.data)
}

// Release returns the buffer to the pool.
func (b *Buffer) Release() {
	if b.pool != nil {
		b.pool.Put(b.data)
	}
}
