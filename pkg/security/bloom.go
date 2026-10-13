package security

import (
	"hash/fnv"
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding atomic counters to 64 bytes.
type CacheLinePad [8]uint64

// BloomFilter implements a lock-free, cache-line padded Bloom filter for IP blacklisting.
type BloomFilter struct {
	bits   []atomic.Uint64
	size   uint64
	hashes uint64
}

// NewBloomFilter creates a new Bloom filter with the specified size (in bits) and number of hash functions.
func NewBloomFilter(size uint64, hashes uint64) *BloomFilter {
	numWords := (size + 63) / 64
	return &BloomFilter{
		bits:   make([]atomic.Uint64, numWords),
		size:   size,
		hashes: hashes,
	}
}

func (b *BloomFilter) getHashes(data []byte) []uint64 {
	h := fnv.New64a()
	h.Write(data)
	base := h.Sum64()

	results := make([]uint64, b.hashes)
	for i := uint64(0); i < b.hashes; i++ {
		results[i] = (base + i*0x9e3779b9) % b.size
	}
	return results
}

// Add inserts an IP into the filter.
func (b *BloomFilter) Add(ip []byte) {
	for _, h := range b.getHashes(ip) {
		wordIdx := h / 64
		bitIdx := h % 64
		b.bits[wordIdx].Or(1 << bitIdx)
	}
}

// Contains checks if an IP is in the filter.
func (b *BloomFilter) Contains(ip []byte) bool {
	for _, h := range b.getHashes(ip) {
		wordIdx := h / 64
		bitIdx := h % 64
		if (b.bits[wordIdx].Load() & (1 << bitIdx)) == 0 {
			return false
		}
	}
	return true
}
