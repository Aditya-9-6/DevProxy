package tdigest

import (
	"math"
	"sort"
	"sync"
)

// Digest implements a cache-aligned, fixed-buffer t-digest quantile estimator.
type Digest struct {
	compression float64
	centroids   []Centroid
	count       uint64
	mu          sync.Mutex
}

var digestPool = sync.Pool{
	New: func() any {
		return New(100)
	},
}

// New creates a new t-digest with the given compression factor.
func New(compression float64) *Digest {
	if compression <= 0 {
		compression = 100
	}
	maxCentroids := int(compression * 3)
	if maxCentroids < 50 {
		maxCentroids = 50
	}
	return &Digest{
		compression: compression,
		centroids:   make([]Centroid, 0, maxCentroids),
	}
}

// AcquireDigest gets a Digest from sync.Pool.
func AcquireDigest() *Digest {
	return digestPool.Get().(*Digest)
}

// Release returns a Digest to sync.Pool after resetting it.
func (d *Digest) Release() {
	d.Reset()
	digestPool.Put(d)
}

// Reset clears the digest state for reuse.
func (d *Digest) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.centroids = d.centroids[:0]
	d.count = 0
}

// Add records a value into the t-digest stream.
func (d *Digest) Add(val float64) {
	d.AddWeighted(val, 1.0)
}

// AddWeighted records a value with a custom weight.
func (d *Digest) AddWeighted(val float64, weight float64) {
	if math.IsNaN(val) || weight <= 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Find closest centroid or append if under capacity
	if len(d.centroids) < cap(d.centroids) {
		d.centroids = append(d.centroids, Centroid{})
		idx := len(d.centroids) - 1
		d.centroids[idx].SetMean(val)
		d.centroids[idx].SetWeight(weight)
		d.count++
		return
	}

	// Simple streaming compression merge if full
	minIdx := 0
	minDist := math.MaxFloat64
	for i := range d.centroids {
		dist := math.Abs(d.centroids[i].GetMean() - val)
		if dist < minDist {
			minDist = dist
			minIdx = i
		}
	}
	d.centroids[minIdx].Add(val, weight)
	d.count++
}

// Quantile calculates the estimated quantile (0.0 to 1.0).
func (d *Digest) Quantile(q float64) float64 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.centroids) == 0 || q < 0 || q > 1 {
		return math.NaN()
	}

	sort.Slice(d.centroids, func(i, j int) bool {
		return d.centroids[i].GetMean() < d.centroids[j].GetMean()
	});

	// Single centroid edge case
	if len(d.centroids) == 1 {
		return d.centroids[0].GetMean()
	}

	totalWeight := 0.0
	for i := range d.centroids {
		totalWeight += d.centroids[i].GetWeight()
	}

	index := q * totalWeight
	sum := 0.0
	for i := range d.centroids {
		w := d.centroids[i].GetWeight()
		if sum+w >= index {
			// Linear interpolation within centroid
			prevSum := sum - (w / 2.0)
			if index < prevSum || i == 0 {
				return d.centroids[i].GetMean()
			}
			prevMean := d.centroids[i-1].GetMean()
			currMean := d.centroids[i].GetMean()
			fr := (index - prevSum) / w
			return prevMean + fr*(currMean-prevMean)
		}
		sum += w
	}

	return d.centroids[len(d.centroids)-1].GetMean()
}

// Count returns total number of observations recorded.
func (d *Digest) Count() uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.count
}
