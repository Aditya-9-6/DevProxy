package security

import (
	"sync/atomic"
)

const (
	fnvOffset64 = 14695981039346656037
	fnvPrime64  = 1099511628211
)

// fnv1a64 computes 64-bit FNV-1a hash inline without heap allocations.
func fnv1a64(data []byte) uint64 {
	h := uint64(fnvOffset64)
	for _, b := range data {
		h ^= uint64(b)
		h *= fnvPrime64
	}
	return h
}

// BloomFilter implements a thread-safe, lock-free Bloom filter for IP blacklisting using atomic bitsets.
// It uses inline double-hashing to avoid heap allocations.
type BloomFilter struct {
	bits   []atomic.Uint64
	size   uint64
	hashes uint64
}

// NewBloomFilter creates a new Bloom filter with the specified size (in bits) and number of hash functions.
// If size is zero, it defaults to 65536.
func NewBloomFilter(size uint64, hashes uint64) *BloomFilter {
	if size == 0 {
		size = 65536
	}
	numWords := (size + 63) / 64
	return &BloomFilter{
		bits:   make([]atomic.Uint64, numWords),
		size:   size,
		hashes: hashes,
	}
}

// Add inserts an IP into the filter using inline double-hashing.
func (b *BloomFilter) Add(ip []byte) {
	if len(ip) == 0 {
		return
	}
	sum := fnv1a64(ip)
	h1 := sum & 0xffffffff
	h2 := sum >> 32
	if h2 == 0 {
		h2 = 1
	}

	for i := uint64(0); i < b.hashes; i++ {
		idx := (h1 + i*h2) % b.size
		wordIdx := idx / 64
		bitIdx := idx % 64
		b.bits[wordIdx].Or(1 << bitIdx)
	}
}

// Contains checks if an IP is in the filter using inline double-hashing.
func (b *BloomFilter) Contains(ip []byte) bool {
	if len(ip) == 0 {
		return false
	}
	sum := fnv1a64(ip)
	h1 := sum & 0xffffffff
	h2 := sum >> 32
	if h2 == 0 {
		h2 = 1
	}

	for i := uint64(0); i < b.hashes; i++ {
		idx := (h1 + i*h2) % b.size
		wordIdx := idx / 64
		bitIdx := idx % 64
		if (b.bits[wordIdx].Load() & (1 << bitIdx)) == 0 {
			return false
		}
	}
	return true
}
