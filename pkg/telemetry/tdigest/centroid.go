package tdigest

import (
	"math"
	"sync/atomic"
)

// Centroid represents a single weighted mean cluster in the t-digest.
// It uses atomic operations for lock-free updates to mean and weight.
type Centroid struct {
	mean   atomic.Uint64 // stored as math.Float64bits(mean)
	weight atomic.Uint64 // stored as math.Float64bits(weight)
}

// GetMean returns the decoded float64 mean.
func (c *Centroid) GetMean() float64 {
	bits := c.mean.Load()
	return math.Float64frombits(bits)
}

// SetMean sets the float64 mean atomically.
func (c *Centroid) SetMean(val float64) {
	c.mean.Store(math.Float64bits(val))
}

// GetWeight returns the decoded float64 weight.
func (c *Centroid) GetWeight() float64 {
	bits := c.weight.Load()
	return math.Float64frombits(bits)
}

// SetWeight sets the float64 weight atomically.
func (c *Centroid) SetWeight(val float64) {
	c.weight.Store(math.Float64bits(val))
}

// Add adds value with given weight to the centroid atomically.
func (c *Centroid) Add(val, w float64) {
	for {
		oldMeanBits := c.mean.Load()
		oldWeightBits := c.weight.Load()
		
		oldMean := math.Float64frombits(oldMeanBits)
		oldWeight := math.Float64frombits(oldWeightBits)
		
		newWeight := oldWeight + w
		var newMean float64
		if newWeight == 0 {
			newMean = 0
		} else {
			newMean = oldMean + (val-oldMean)*(w/newWeight)
		}
		
		if c.mean.CompareAndSwap(oldMeanBits, math.Float64bits(newMean)) &&
		   c.weight.CompareAndSwap(oldWeightBits, math.Float64bits(newWeight)) {
			break
		}
	}
}
