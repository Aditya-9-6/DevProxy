package congestion

import (
	"crypto/rand"
	"math/big"
	"sync/atomic"
)

// REDController implements a lock-free Random Early Drop congestion controller.
// It uses atomic counters to track queue occupancy and calculates drop probability
// based on configured thresholds without mutex contention.
type REDController struct {
	// occupancy tracks the current number of items in the queue.
	occupancy atomic.Int64
	// minThreshold is the occupancy level below which no packets are dropped.
	minThreshold int64
	// maxThreshold is the occupancy level above which all packets are dropped.
	maxThreshold int64
	// maxProb is the maximum drop probability (0-1000).
	maxProb int64
}

// NewREDController creates a new RED controller.
func NewREDController(min, max int64, maxProb int64) *REDController {
	return &REDController{
		minThreshold: min,
		maxThreshold: max,
		maxProb:      maxProb,
	}
}

// ShouldDrop returns true if the packet should be dropped based on current occupancy.
func (r *REDController) ShouldDrop() bool {
	occ := r.occupancy.Load()
	if occ < r.minThreshold {
		return false
	}
	if occ >= r.maxThreshold {
		return true
	}

	// Calculate probability: (occ - min) / (max - min) * maxProb
	prob := big.NewInt(0)
	prob.Sub(big.NewInt(occ), big.NewInt(r.minThreshold))
	prob.Mul(prob, big.NewInt(r.maxProb))
	prob.Div(prob, big.NewInt(r.maxThreshold-r.minThreshold))

	// Generate a cryptographically secure random number between 0 and 1000
	random, _ := rand.Int(rand.Reader, big.NewInt(1000))
	return prob.Cmp(random) >= 0
}

// Increment increases the occupancy count.
func (r *REDController) Increment() {
	r.occupancy.Add(1)
}

// Decrement decreases the occupancy count.
func (r *REDController) Decrement() {
	r.occupancy.Add(-1)
}
