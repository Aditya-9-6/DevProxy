package tdigest

// Centroid represents a cluster in the t-digest.
// Padded to 64 bytes to prevent false sharing on CPU caches.
type Centroid struct {
	Mean   uint64   // Scaled mean value
	Weight uint64   // Weight of the centroid
	_      [48]byte // Padding for cache-line alignment
}

// NewCentroid creates a new centroid with initial values.
func NewCentroid(mean, weight uint64) *Centroid {
	return &Centroid{
		Mean:   mean,
		Weight: weight,
	}
}
