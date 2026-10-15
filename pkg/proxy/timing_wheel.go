package proxy

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	WheelSize = 1024
	TickMs    = 100
	Levels    = 3
	LevelBits = 10
)

type wheelBucket struct {
	_     [56]byte
	count atomic.Uint64
	_     [56]byte
}

type TimingWheel struct {
	mu      sync.Mutex
	buckets [Levels][WheelSize]wheelBucket
	cursor  atomic.Uint64
}

func NewTimingWheel() *TimingWheel {
	return &TimingWheel{}
}

func (tw *TimingWheel) Register(deadline time.Duration) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	ticks := uint64(deadline.Milliseconds() / TickMs)
	if ticks == 0 {
		ticks = 1
	}
	cur := tw.cursor.Load()
	expiresAt := cur + ticks

	// Hierarchical placement across levels
	for i := 0; i < Levels; i++ {
		shift := uint(i * LevelBits)
		mask := uint64(WheelSize - 1)
		idx := (expiresAt >> shift) & mask
		if i == Levels-1 || (expiresAt>>(shift+LevelBits)) == (cur>>(shift+LevelBits)) {
			tw.buckets[i][idx].count.Add(1)
			break
		}
	}
}

func (tw *TimingWheel) Tick() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	// Safely increment cursor without integer overflow issues on wrap
	cur := tw.cursor.Add(1)
	if cur == 0xFFFFFFFFFFFFFFFF {
		tw.cursor.Store(0)
	}
}

func (tw *TimingWheel) GetExpiredCount() uint64 {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	cur := tw.cursor.Load()
	idx := cur % WheelSize
	count := tw.buckets[0][idx].count.Swap(0)

	// Cascade elements from higher levels if cursor wrapped around level boundary
	for i := 1; i < Levels; i++ {
		shift := uint(i * LevelBits)
		if (cur >> shift) == 0 {
			hIdx := (cur >> (shift - LevelBits)) % WheelSize
			c := tw.buckets[i][hIdx].count.Swap(0)
			if c > 0 {
				tw.buckets[0][(idx+1)%WheelSize].count.Add(c)
			}
		}
	}

	return count
}
