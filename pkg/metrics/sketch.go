package metrics

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"
)

const (
	cacheLineSize               = 64
	defaultCountMinSketchWindow = time.Minute
)

// paddedCounter stores an atomic counter in a cache-line-sized value to reduce false sharing.
type paddedCounter struct {
	val uint64
	_   [cacheLineSize - 8]byte
}

// CountMinSketch is a lock-free, cache-aligned Count-Min Sketch for frequency estimation.
// It uses pre-allocated fixed arrays of atomic counters and MurmurHash3 hashing.
type CountMinSketch struct {
	width    uint64
	depth    uint64
	counters [][]paddedCounter
}

// NewCountMinSketch creates a new CountMinSketch with the given width (number of columns)
// and depth (number of hash functions/rows). Zero dimensions are normalized to one.
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
// It is safe for concurrent use. Non-positive counts are ignored because the
// sketch stores unsigned frequency counters.
func (cms *CountMinSketch) Add(key []byte, count int64) {
	if cms == nil || count <= 0 || cms.width == 0 || cms.depth == 0 {
		return
	}

	h1 := murmurHash64(key, 0)
	h2 := murmurHash64(key, 1)
	// Every positive int64 is representable as uint64.
	increment := uint64(count) // #nosec G115
	for i := uint64(0); i < cms.depth; i++ {
		idx := (h1 + i*h2) % cms.width
		atomic.AddUint64(&cms.counters[i][idx].val, increment)
	}
}

// Estimate returns the estimated frequency of the given key.
// It is safe for concurrent use.
func (cms *CountMinSketch) Estimate(key []byte) uint64 {
	if cms == nil || cms.width == 0 || cms.depth == 0 {
		return 0
	}

	h1 := murmurHash64(key, 0)
	h2 := murmurHash64(key, 1)
	var minimum uint64
	for i := uint64(0); i < cms.depth; i++ {
		idx := (h1 + i*h2) % cms.width
		value := atomic.LoadUint64(&cms.counters[i][idx].val)
		if i == 0 || value < minimum {
			minimum = value
		}
	}
	return minimum
}

// Reset clears all counters in the sketch.
func (cms *CountMinSketch) Reset() {
	if cms == nil {
		return
	}

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

	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewSlidingWindowCountMinSketch creates a new SlidingWindowCountMinSketch with the given
// width, depth, and window duration. Non-positive windows use a safe default duration.
func NewSlidingWindowCountMinSketch(width, depth uint64, window time.Duration) *SlidingWindowCountMinSketch {
	if window <= 0 {
		window = defaultCountMinSketchWindow
	}

	s := &SlidingWindowCountMinSketch{
		inactive: NewCountMinSketch(width, depth),
		window:   window,
	}
	s.active.Store(NewCountMinSketch(width, depth))
	return s
}

// Start begins the background goroutine that periodically swaps the active sketch
// and resets the inactive one. It respects ctx cancellation and cleans up on Stop.
// Calling Start while an instance is already running has no effect.
func (s *SlidingWindowCountMinSketch) Start(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if s.cancel != nil {
		return
	}
	if s.window <= 0 {
		s.window = defaultCountMinSketchWindow
	}
	if s.active.Load() == nil {
		s.active.Store(NewCountMinSketch(1, 1))
	}
	if s.inactive == nil {
		s.inactive = NewCountMinSketch(1, 1)
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	go s.run(runCtx, done)
}

// Stop halts the background goroutine and waits for it to finish.
// It is safe to call Stop more than once or concurrently with Start.
func (s *SlidingWindowCountMinSketch) Stop() {
	if s == nil {
		return
	}

	s.lifecycleMu.Lock()
	cancel := s.cancel
	done := s.done
	s.lifecycleMu.Unlock()

	if cancel == nil {
		return
	}

	cancel()
	<-done

	s.lifecycleMu.Lock()
	if s.done == done {
		s.cancel = nil
		s.done = nil
	}
	s.lifecycleMu.Unlock()
}

func (s *SlidingWindowCountMinSketch) run(ctx context.Context, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(s.window)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.rotate()
		}
	}
}

func (s *SlidingWindowCountMinSketch) rotate() {
	s.inactive.Reset()
	oldActive := s.active.Swap(s.inactive)
	s.inactive = oldActive
}

// Add increments the count for the given key in the active sketch.
func (s *SlidingWindowCountMinSketch) Add(key []byte, count int64) {
	if s == nil {
		return
	}
	active := s.active.Load()
	if active == nil {
		return
	}
	active.Add(key, count)
}

// Estimate returns the estimated frequency of the given key from the active sketch.
func (s *SlidingWindowCountMinSketch) Estimate(key []byte) uint64 {
	if s == nil {
		return 0
	}
	active := s.active.Load()
	if active == nil {
		return 0
	}
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

	for i := 0; i < len(key)/8; i++ {
		k := binary.LittleEndian.Uint64(key[i*8:])
		k *= m
		k ^= k >> r
		k *= m
		h ^= k
		h *= m
	}

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
