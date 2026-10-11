package security

import (
	"hash/fnv"
	"sync/atomic"
)

// BloomFilter implements a thread-safe, lock-free Bloom filter using atomic bitsets.
type BloomFilter struct {
	bits []atomic.Uint64
	size uint64
	k    uint64
}

// NewBloomFilter creates a new filter with m bits and k hash functions.
func NewBloomFilter(m uint64, k uint64) *BloomFilter {
	numWords := (m + 63) / 64
	return &BloomFilter{
		bits: make([]atomic.Uint64, numWords),
		size: m,
		k:    k,
	}
}

// hash computes the i-th hash using double hashing with FNV-1a.
func (b *BloomFilter) hash(data []byte, i uint64) uint64 {
	h := fnv.New64a()
	h.Write(data)
	sum := h.Sum64()
	// Double hashing: (h1 + i * h2) % size
	// We use the upper and lower 32 bits of the 64-bit hash as h1 and h2
	h1 := sum & 0xffffffff
	h2 := sum >> 32
	return (h1 + i*h2) % b.size
}

// Add inserts an item into the filter.
func (b *BloomFilter) Add(data []byte) {
	for i := uint64(0); i < b.k; i++ {
		idx := b.hash(data, i)
		word := idx / 64
		bit := idx % 64
		b.bits[word].Or(1 << bit)
	}
}

// Contains checks if an item is in the filter.
func (b *BloomFilter) Contains(data []byte) bool {
	for i := uint64(0); i < b.k; i++ {
		idx := b.hash(data, i)
		word := idx / 64
		bit := idx % 64
		if b.bits[word].Load()&(1<<bit) == 0 {
			return false
		}
	}
	return true
}
