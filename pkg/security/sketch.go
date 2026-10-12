package security

import (
	"context"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

const (
	SketchWidth = 1024
	SketchDepth = 4
)

type CountMinSketch struct {
	facets  [SketchDepth]maphash.Seed
	buckets [SketchDepth][SketchWidth]atomic.Uint64
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewCountMinSketch() *CountMinSketch {
	ctx, cancel := context.WithCancel(context.Background())
	c := &CountMinSketch{
		cancel: cancel,
	}
	for i := 0; i < SketchDepth; i++ {
		c.facets[i] = maphash.MakeSeed()
	}
	c.wg.Add(1)
	go c.decayLoop(ctx)
	return c
}

func (c *CountMinSketch) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

func (c *CountMinSketch) decayLoop(ctx context.Context) {
	defer c.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < SketchDepth; i++ {
				for j := 0; j < SketchWidth; j++ {
					val := c.buckets[i][j].Load()
					if val > 0 {
						c.buckets[i][j].Store(val / 2)
					}
				}
			}
		}
	}
}

func (c *CountMinSketch) hash(data []byte, depth int) uint64 {
	var h maphash.Hash
	h.SetSeed(c.facets[depth])
	h.Write(data)
	return h.Sum64() % SketchWidth
}

func (c *CountMinSketch) Increment(key []byte) {
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, i)
		c.buckets[i][h].Add(1)
	}
}

func (c *CountMinSketch) Estimate(key []byte) uint64 {
	min := uint64(^uint64(0))
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, i)
		val := c.buckets[i][h].Load()
		if val < min {
			min = val
		}
	}
	return min
}
