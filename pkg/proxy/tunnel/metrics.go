package tunnel

import (
	"sync/atomic"
)

// TunnelMetrics tracks zero-copy encapsulated packet throughput and drops using lock-free atomic counters.
type TunnelMetrics struct {
	encapsulatedPackets atomic.Uint64
	decapsulatedPackets atomic.Uint64
	encapsulatedBytes   atomic.Uint64
	decapsulatedBytes   atomic.Uint64
	droppedPackets      atomic.Uint64
}

var globalMetrics = &TunnelMetrics{}

// GetGlobalMetrics returns the singleton tunnel metrics tracker.
func GetGlobalMetrics() *TunnelMetrics {
	return globalMetrics
}

func (m *TunnelMetrics) IncEncapsulated(bytesLen int) {
	m.encapsulatedPackets.Add(1)
	m.encapsulatedBytes.Add(uint64(bytesLen))
}

func (m *TunnelMetrics) IncDecapsulated(bytesLen int) {
	m.decapsulatedPackets.Add(1)
	m.decapsulatedBytes.Add(uint64(bytesLen))
}

func (m *TunnelMetrics) IncDropped() {
	m.droppedPackets.Add(1)
}

func (m *TunnelMetrics) Stats() (encPkts, decPkts, encBytes, decBytes, dropped uint64) {
	return m.encapsulatedPackets.Load(),
		m.decapsulatedPackets.Load(),
		m.encapsulatedBytes.Load(),
		m.decapsulatedBytes.Load(),
		m.droppedPackets.Load()
}
