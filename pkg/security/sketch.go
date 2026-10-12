package security

import (
	"hash/maphash"
	"sync/atomic"
	"time"
)

const (
	SketchWidth = 1024
	SketchDepth = 4
)

type CountMinSketch struct {
	buckets [SketchDepth][SketchWidth]atomic.Uint64
	hashers [SketchDepth]maphash.Hash
}

func NewCountMinSketch() *CountMinSketch {
	c := &CountMinSketch{}
	for i := 0; i < SketchDepth; i++ {
		c.hashers[i].SetSeed(maphash.MakeSeed())
	}
	go c.decayLoop()
	return c
}

func (c *CountMinSketch) decayLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
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

func (c *CountMinSketch) hash(data []byte, depth int) uint64 {
	c.hashers[depth].Reset()
	c.hashers[depth].Write(data)
	return c.hashers[depth].Sum64() % SketchWidth
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
