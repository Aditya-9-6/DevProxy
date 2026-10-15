package tdigest

import (
	"sync"
)

// Digest implements a fixed-size t-digest for quantile estimation.
type Digest struct {
	centroids []*Centroid
	maxSize   int
	pool      sync.Pool
}

// NewDigest creates a new t-digest with a fixed number of centroids.
func NewDigest(maxSize int) *Digest {
	d := &Digest{
		centroids: make([]*Centroid, maxSize),
		maxSize:   maxSize,
	}
	for i := 0; i < maxSize; i++ {
		d.centroids[i] = NewCentroid(0, 0)
	}
	return d
}

// Add records a new observation into the digest.
func (d *Digest) Add(value uint64) {
	// Simple round-robin or hash-based distribution for lock-free streaming
	idx := value % uint64(d.maxSize)
	c := d.centroids[idx]
	c.Mean.Store(value)
	c.Weight.Add(1)
}

// GetQuantile estimates the value at the given quantile (0.0-1.0).
func (d *Digest) GetQuantile(q float64) uint64 {
	var totalWeight uint64
	for _, c := range d.centroids {
		totalWeight += c.Weight.Load()
	}
	if totalWeight == 0 {
		return 0
	}
	target := uint64(float64(totalWeight) * q)
	var currentWeight uint64
	for _, c := range d.centroids {
		w := c.Weight.Load()
		if currentWeight+w >= target {
			return c.Mean.Load()
		}
		currentWeight += w
	}
	return 0
}
