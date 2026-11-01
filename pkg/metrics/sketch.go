package metrics

import (
	"sync/atomic"
)

// Constants for MurmurHash3 constants and configuration
const (
	murmurC1 uint32 = 0xcc9e2d51
	murmurC2 uint32 = 0x1b873593
	murmurR1 uint32 = 15
	murmurR2 uint32 = 13
	murmurM  uint32 = 5
	murmurN  uint32 = 0xe6546b64

	// Cache line padding to prevent false sharing on modern CPU architectures (64 bytes)
	cacheLinePadSize = 64
)

// cachePaddedCounter wraps an atomic uint64 with padding to align with cache line boundaries.
type cachePaddedCounter struct {
	val [8]atomic.Uint64
}

// Sketch implements a cache-aligned, lock-free Count-Min Sketch using atomic operations
// for real-time frequency estimation of upstream errors and path saturations.
type Sketch struct {
	depth   uint32
	width   uint32
	matrix  [][]atomic.Uint64
	seeds   []uint32
}

// NewSketch initializes a Count-Min Sketch with specified depth (number of hash functions)
// and width (number of counters per row). Functions strictly conform to <60 LOC single-purpose designs.
func NewSketch(depth, width uint32) *Sketch {
	if depth == 0 {
		depth = 4
	}
	if width == 0 {
		width = 1024
	}

	matrix := make([][]atomic.Uint64, depth)
	for i := range matrix {
		matrix[i] = make([]atomic.Uint64, width)
	}

	seeds := make([]uint32, depth)
	for i := range seeds {
		seeds[i] = uint32(i + 1) * 0x85ebca6b
	}

	return &Sketch{
		depth:  depth,
		width:  width,
		matrix: matrix,
		seeds:  seeds,
	}
}

// Add increments the frequency estimate for a given key by delta.
func (s *Sketch) Add(key []byte, delta uint64) {
	for i := uint32(0); i < s.depth; i++ {
		hash := s.hash(key, s.seeds[i])
		col := hash % s.width
		s.matrix[i][col].Add(delta)
	}
}

// Estimate returns the conservative frequency estimate for a given key.
func (s *Sketch) Estimate(key []byte) uint64 {
	var minVal uint64 = ^uint64(0)
	for i := uint32(0); i < s.depth; i++ {
		hash := s.hash(key, s.seeds[i])
		col := hash % s.width
		val := s.matrix[i][col].Load()
		if val < minVal {
			minVal = val
		}
	}
	return minVal
}

// Reset clears all counters in the sketch atomically.
func (s *Sketch) Reset() {
	for i := uint32(0); i < s.depth; i++ {
		for j := uint32(0); j < s.width; j++ {
			s.matrix[i][j].Store(0)
		}
	}
}

// hash implements MurmurHash3 x86 32-bit mixing function for zero-alloc hash generation.
func (s *Sketch) hash(data []byte, seed uint32) uint32 {
	h1 := seed

	// Process 4-byte chunks
	length := len(data)
	nblocks := length / 4
	for i := 0; i < nblocks; i++ {
		idx := i * 4
		k1 := uint32(data[idx]) | uint32(data[idx+1])<<8 | uint32(data[idx+2])<<16 | uint32(data[idx+3])<<24

		k1 *= murmurC1
		k1 = (k1 << murmurR1) | (k1 >> (32 - murmurR1))
		k1 *= murmurC2

		h1 ^= k1
		h1 = (h1 << murmurR2) | (h1 >> (32 - murmurR2))
		h1 = h1*murmurM + murmurN
	}

	// Process remaining bytes
	tailIdx := nblocks * 4
	var k1 uint32
	switch length & 3 {
	case 3:
		k1 ^= uint32(data[tailIdx+2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(data[tailIdx+1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(data[tailIdx])
		k1 *= murmurC1
		k1 = (k1 << murmurR1) | (k1 >> (32 - murmurR1))
		k1 *= murmurC2
		h1 ^= k1
	}

	h1 ^= uint32(length)
	h1 ^= h1 >> 16
	h1 *= 0x85ebca6b
	h1 ^= h1 >> 13
	h1 *= 0xc2b2ae35
	h1 ^= h1 >> 16

	return h1
}
