package http3

import (
	"sync"
)

// DatagramBuffer represents a pre-allocated buffer for QUIC datagrams.
var DatagramPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1500)
	},
}

// Scheduler handles multi-path datagram dispatching.
type Scheduler struct {
	tracker *MigrationTracker
}

func NewScheduler(mt *MigrationTracker) *Scheduler {
	return &Scheduler{tracker: mt}
}

// Schedule dispatches a datagram to the best available path based on atomic metrics.
func (s *Scheduler) Schedule(data []byte) int {
	bestPath := 0
	minLatency := ^uint64(0)

	for i, path := range s.tracker.Paths {
		if !path.Active.Load() {
			continue
		}
		lat := path.LatencyNs.Load()
		if lat < minLatency {
			minLatency = lat
			bestPath = i
		}
	}
	return bestPath
}
