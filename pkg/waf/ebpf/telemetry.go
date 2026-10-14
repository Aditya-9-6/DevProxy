package ebpf

import "sync/atomic"

// CacheLinePad prevents false sharing on hot metric counters.
type CacheLinePad [8]uint64

// Telemetry tracks kernel-level security events using atomic counters.
type Telemetry struct {
	DropCount CacheLinePad
	HitCount  CacheLinePad
}

func (t *Telemetry) RecordDrop() {
	atomic.AddUint64(&t.DropCount[0], 1)
}

func (t *Telemetry) RecordHit() {
	atomic.AddUint64(&t.HitCount[0], 1)
}

func (t *Telemetry) GetStats() (uint64, uint64) {
	return atomic.LoadUint64(&t.DropCount[0]), atomic.LoadUint64(&t.HitCount[0])
}
