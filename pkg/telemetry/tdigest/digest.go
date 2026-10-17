package tdigest

import (
	"sort"
	"sync"
)

// Digest implements a fixed-size t-digest for quantile estimation.
type Digest struct {
	mu        sync.Mutex
	centroids []*Centroid
	maxSize   int
	pool      sync.Pool
}

// NewDigest creates a new t-digest with a fixed number of centroids.
func NewDigest(maxSize int) *Digest {
	if maxSize <= 0 {
		maxSize = 100
	}
	d := &Digest{
		maxSize: maxSize,
	}
	d.pool.New = func() any {
		return make([]*Centroid, 0, maxSize)
	}
	d.centroids = make([]*Centroid, 0, maxSize)
	return d
}

// Add records a new observation into the digest with centroid compression.
func (d *Digest) Add(value uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add as a new single-weight centroid
	d.centroids = append(d.centroids, NewCentroid(value, 1))

	// Compress if we exceed capacity
	if len(d.centroids) > d.maxSize*2 {
		d.compress()
	}
}

// compress merges nearby centroids to maintain the size bound.
func (d *Digest) compress() {
	if len(d.centroids) <= 1 {
		return
	}

	// Sort centroids by mean
	sort.Slice(d.centroids, func(i, j int) bool {
		return d.centroids[i].Mean < d.centroids[j].Mean
	})

	pivots, ok := d.pool.Get().([]*Centroid)
	if !ok {
		pivots = make([]*Centroid, 0, d.maxSize)
	} else {
		pivots = pivots[:0]
	}

	curr := d.centroids[0]
	for i := 1; i < len(d.centroids); i++ {
		next := d.centroids[i]
		// If current centroid and next can be merged without exceeding scale or max size constraints
		if len(pivots) < d.maxSize-1 && (curr.Weight+next.Weight <= uint64(d.maxSize)) {
			totalW := curr.Weight + next.Weight
			if totalW > 0 {
				curr.Mean = (curr.Mean*curr.Weight + next.Mean*next.Weight) / totalW
				curr.Weight = totalW
			}
		} else {
			pivots = append(pivots, curr)
			curr = next
		}
	}
	pivots = append(pivots, curr)

	d.centroids = pivots
}

// GetQuantile estimates the value at the given quantile (0.0-1.0).
func (d *Digest) GetQuantile(q float64) uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.centroids) == 0 {
		return 0
	}

	// Ensure centroids are sorted before quantile estimation
	sort.Slice(d.centroids, func(i, j int) bool {
		return d.centroids[i].Mean < d.centroids[j].Mean
	})

	if q <= 0.0 {
		return d.centroids[0].Mean
	}
	if q >= 1.0 {
		return d.centroids[len(d.centroids)-1].Mean
	}

	var totalWeight uint64
	for _, c := range d.centroids {
		totalWeight += c.Weight
	}
	if totalWeight == 0 {
		return 0
	}

	target := uint64(float64(totalWeight) * q)
	var currentWeight uint64
	for _, c := range d.centroids {
		w := c.Weight
		if currentWeight+w >= target {
			return c.Mean
		}
		currentWeight += w
	}

	return d.centroids[len(d.centroids)-1].Mean
}
