package security

import (
	"hash/fnv"
	"sync/atomic"
)

// CacheLinePad prevents false sharing on hot atomic counters.
type CacheLinePad [8]uint64

// CookieFilter implements a lock-free probabilistic signature verification filter.
// It uses a bit-array of size 2^20 (1M bits) for low collision probability.
type CookieFilter struct {
	bits [1024 * 1024 / 64]CacheLinePad
}

func NewCookieFilter() *CookieFilter {
	return &CookieFilter{}
}

// Add inserts a signature into the filter.
func (f *CookieFilter) Add(signature string) {
	h1, h2 := f.hashes(signature)
	f.setBit(h1)
	f.setBit(h2)
}

// Contains checks if a signature might exist in the filter.
func (f *CookieFilter) Contains(signature string) bool {
	h1, h2 := f.hashes(signature)
	return f.getBit(h1) && f.getBit(h2)
}

func (f *CookieFilter) hashes(s string) (uint64, uint64) {
	h := fnv.New64a()
	h.Write([]byte(s))
	h1 := h.Sum64()
	h.Write([]byte("salt"))
	h2 := h.Sum64()
	return h1 % (1024 * 1024), h2 % (1024 * 1024)
}

func (f *CookieFilter) setBit(idx uint64) {
	word := idx / 64
	bit := idx % 64
	atomic.OrUint64(&f.bits[word][0], 1<<bit)
}

func (f *CookieFilter) getBit(idx uint64) bool {
	word := idx / 64
	bit := idx % 64
	return (atomic.LoadUint64(&f.bits[word][0]) & (1 << bit)) != 0
}
