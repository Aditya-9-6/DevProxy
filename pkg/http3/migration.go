package http3

import (
	"sync/atomic"
)

// PathState tracks the migration status of a QUIC connection path.
// Uses cache-line padded atomic counters to prevent false sharing.
type PathState struct {
	_          [56]byte // Padding for cache line alignment
	Active     atomic.Bool
	PacketLoss atomic.Uint64
	LatencyNs  atomic.Uint64
	_          [56]byte
}

// MigrationTracker manages multiple paths for a single QUIC connection.
type MigrationTracker struct {
	Paths [4]*PathState
}

func NewMigrationTracker() *MigrationTracker {
	mt := &MigrationTracker{}
	for i := range mt.Paths {
		mt.Paths[i] = &PathState{}
	}
	return mt
}

func (mt *MigrationTracker) UpdatePathMetrics(pathIdx int, loss uint64, latency uint64) {
	if pathIdx < 0 || pathIdx >= len(mt.Paths) {
		return
	}
	mt.Paths[pathIdx].PacketLoss.Store(loss)
	mt.Paths[pathIdx].LatencyNs.Store(latency)
}
