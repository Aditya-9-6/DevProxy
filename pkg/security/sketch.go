package security

import (
	"hash/fnv"
	"sync/atomic"
)

const (
	SketchWidth = 1024
	SketchDepth = 4
)

type CountMinSketch struct {
	buckets [SketchDepth][SketchWidth]atomic.Uint64
}

func NewCountMinSketch() *CountMinSketch {
	return &CountMinSketch{}
}

func (c *CountMinSketch) hash(data []byte, seed uint64) uint64 {
	h := fnv.New64a()
	h.Write(data)
	return (h.Sum64() + seed) % SketchWidth
}

func (c *CountMinSketch) Increment(key []byte) {
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, uint64(i))
		c.buckets[i][h].Add(1)
	}
}

func (c *CountMinSketch) Estimate(key []byte) uint64 {
	min := uint64(^uint64(0))
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, uint64(i))
		val := c.buckets[i][h].Load()
		if val < min {
			min = val
		}
	}
	return min
}
