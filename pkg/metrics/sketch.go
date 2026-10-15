package metrics

import (
	"hash/fnv"
	"sync/atomic"
)

const (
	SketchWidth = 1024
	SketchDepth = 4
)

type CacheLinePad [8]uint64

type CountMinSketch struct {
	table [SketchDepth][SketchWidth]atomic.Uint64
}

func NewCountMinSketch() *CountMinSketch {
	return &CountMinSketch{}
}

func (c *CountMinSketch) hash(key string, i int) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	h.Write([]byte{byte(i)})
	return h.Sum32() % SketchWidth
}

func (c *CountMinSketch) Increment(key string) {
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, i)
		c.table[i][h].Add(1)
	}
}

func (c *CountMinSketch) Estimate(key string) uint64 {
	min := uint64(^uint64(0))
	for i := 0; i < SketchDepth; i++ {
		h := c.hash(key, i)
		val := c.table[i][h].Load()
		if val < min {
			min = val
		}
	}
	return min
}
