package metrics

import (
	"context"
	"encoding/binary"
	"math"
	"sync/atomic"
	"time"
)

const cacheLineSize = 64

// paddedCounter is an atomic counter aligned to a cache line to prevent false sharing.
type paddedCounter struct {
	val uint64
	// padding to fill the rest of the cache line (64 bytes total)
	_ [cacheLineSize - 8]byte
}

// CountMinSketch is a lock-free, cache-aligned Count-Min Sketch for frequency estimation.
// It uses pre-allocated fixed arrays of atomic counters and MurmurHash3 hashing.
type CountMinSketch struct {
	width    uint64
	depth    uint64
	counters [][]paddedCounter
}

// NewCountMinSketch creates a new CountMinSketch with the given width (number of columns)
// and depth (number of hash functions/rows).
func NewCountMinSketch(width, depth uint64) *CountMinSketch {
	if width == 0 {
		width = 1
	}
	if depth == 0 {
		depth = 1
	}
	cms := &CountMinSketch{
		width:    width,
		depth:    depth,
		counters: make([][]paddedCounter, depth),
	}
	for i := range cms.counters {
		cms.counters[i] = make([]paddedCounter, width)
	}
	return cms
}

// Add increments the count for the given key by the specified count.
// It is safe for concurrent use.
func (cms *CountMinSketch) Add(key []byte, count int64) {
	h1 := murmurHash64(key, 0)
	h2 := murmurHash64(key, 1)
	for i := uint64(0); i < cms.depth; i++ {
		idx := (h1 + i*h2) % cms.width
		atomic.AddUint64(&cms.counters[i][idx].val, uint64(count))
	}
}

// Estimate returns the estimated frequency of the given key.
// It is safe for concurrent use.
func (cms *CountMinSketch) Estimate(key []byte) uint64 {
	h1 := murmurHash64(key, 0)
	h2 := murmurHash64(key, 1)
	var min uint64 = math.MaxUint64
	for i := uint64(0); i < cms.depth; i++ {
		idx := (h1 + i*h2) % cms.width
		val := atomic.LoadUint64(&cms.counters[i][idx].val)
		if val < min {
			min = val
		}
	}
	if min == math.MaxUint64 {
		return 0
	}
	return min
}

// Reset clears all counters in the sketch.
func (cms *CountMinSketch) Reset() {
	for i := range cms.counters {
		for j := range cms.counters[i] {
			atomic.StoreUint64(&cms.counters[i][j].val, 0)
		}
	}
}

// SlidingWindowCountMinSketch is a Count-Min Sketch that automatically resets
// at a fixed interval, providing a sliding window of frequency estimation.
type SlidingWindowCountMinSketch struct {
	active   atomic.Pointer[CountMinSketch]
	inactive *CountMinSketch
	window   time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewSlidingWindowCountMinSketch creates a new SlidingWindowCountMinSketch with the given
// width, depth, and window duration.
func NewSlidingWindowCountMinSketch(width, depth uint64, window time.Duration) *SlidingWindowCountMinSketch {
	cms1 := NewCountMinSketch(width, depth)
	cms2 := NewCountMinSketch(width, depth)
	s := &SlidingWindowCountMinSketch{
		inactive: cms2,
		window:   window,
	}
	s.active.Store(cms1)
	return s
}

// Start begins the background goroutine that periodically swaps the active sketch
// and resets the inactive one. It respects ctx cancellation and cleans up on Stop.
func (s *SlidingWindowCountMinSketch) Start(ctx context.Context) {
	if s.cancel != nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.window)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Reset the sketch that will become active
				s.inactive.Reset()
				oldActive := s.active.Swap(s.inactive)
				s.inactive = oldActive
			}
		}
	}()
}

// Stop halts the background goroutine and waits for it to finish.
func (s *SlidingWindowCountMinSketch) Stop() {
	if s.cancel != nil {
		s.cancel()
		<-s.done
		s.cancel = nil
	}
}

// Add increments the count for the given key in the active sketch.
func (s *SlidingWindowCountMinSketch) Add(key []byte, count int64) {
	active := s.active.Load()
	active.Add(key, count)
}

// Estimate returns the estimated frequency of the given key from the active sketch.
func (s *SlidingWindowCountMinSketch) Estimate(key []byte) uint64 {
	active := s.active.Load()
	return active.Estimate(key)
}

// murmurHash64 computes a 64-bit MurmurHash3 hash of the key with the given seed.
func murmurHash64(key []byte, seed uint64) uint64 {
	const (
		m uint64 = 0xc6bc279692b5c323
		r int    = 47
	)
	h := seed
	n := uint64(len(key))

	// body
	for i := 0; i < len(key)/8; i++ {
		k := binary.LittleEndian.Uint64(key[i*8:])
		k *= m
		k ^= k >> r
		k *= m
		h ^= k
		h *= m
	}

	// tail
	tail := len(key) % 8
	if tail > 0 {
		var k2 uint64
		switch tail {
		case 7:
			k2 ^= uint64(key[len(key)-7]) << 48
			fallthrough
		case 6:
			k2 ^= uint64(key[len(key)-6]) << 40
			fallthrough
		case 5:
			k2 ^= uint64(key[len(key)-5]) << 32
			fallthrough
		case 4:
			k2 ^= uint64(key[len(key)-4]) << 24
			fallthrough
		case 3:
			k2 ^= uint64(key[len(key)-3]) << 16
			fallthrough
		case 2:
			k2 ^= uint64(key[len(key)-2]) << 8
			fallthrough
		case 1:
			k2 ^= uint64(key[len(key)-1])
			k2 *= m
			k2 ^= k2 >> r
			k2 *= m
			h ^= k2
			h *= m
		}
	}

	h ^= n
	h ^= h >> r
	h *= m
	h ^= h >> r
	return h
}
