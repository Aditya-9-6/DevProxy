package tdigest

import "sync/atomic"

// Centroid represents a cluster in the t-digest.
// Padded to 64 bytes to prevent false sharing on CPU caches.
type Centroid struct {
	Mean   atomic.Uint64 // Scaled mean value
	Weight atomic.Uint64 // Weight of the centroid
	_      [48]byte      // Padding for cache-line alignment
}

// NewCentroid creates a new centroid with initial values.
func NewCentroid(mean, weight uint64) *Centroid {
	c := &Centroid{}
	c.Mean.Store(mean)
	c.Weight.Store(weight)
	return c
}
