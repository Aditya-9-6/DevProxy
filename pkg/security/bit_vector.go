package security

import (
	"sync/atomic"
)

// CacheAlignedBitVector provides a lock-free bit-vector backed by atomic uint64 words
// designed for cache-friendly concurrent state transitions.
type CacheAlignedBitVector struct {
	// Padding/alignment structure can be added if needed, but atomic operations on []atomic.Uint64 provide clean word-level access.
	words []atomic.Uint64
	size  int
}

// NewCacheAlignedBitVector creates a bit vector of the specified size in bits.
func NewCacheAlignedBitVector(size int) *CacheAlignedBitVector {
	numWords := (size + 63) / 64
	return &CacheAlignedBitVector{
		words: make([]atomic.Uint64, numWords),
		size:  size,
	}
}

// Set sets the bit at index to 1 atomically.
func (b *CacheAlignedBitVector) Set(index int) {
	if index < 0 || index >= b.size {
		return
	}
	wordIdx := index / 64
	bitIdx := uint(index % 64)
	mask := uint64(1) << bitIdx

	for {
		old := b.words[wordIdx].Load()
		if (old & mask) == mask {
			return
		}
		if b.words[wordIdx].CompareAndSwap(old, old|mask) {
			return
		}
	}
}

// Clear sets the bit at index to 0 atomically.
func (b *CacheAlignedBitVector) Clear(index int) {
	if index < 0 || index >= b.size {
		return
	}
	wordIdx := index / 64
	bitIdx := uint(index % 64)
	mask := ^(uint64(1) << bitIdx)

	for {
		old := b.words[wordIdx].Load()
		if (old & ^mask) == 0 {
			return
		}
		if b.words[wordIdx].CompareAndSwap(old, old&mask) {
			return
		}
	}
}

// Test checks if the bit at index is set (1).
func (b *CacheAlignedBitVector) Test(index int) bool {
	if index < 0 || index >= b.size {
		return false
	}
	wordIdx := index / 64
	bitIdx := uint(index % 64)
	mask := uint64(1) << bitIdx
	return (b.words[wordIdx].Load() & mask) != 0
}
