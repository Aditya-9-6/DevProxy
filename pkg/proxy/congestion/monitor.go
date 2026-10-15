package congestion

import (
	"sync/atomic"
)

// Monitor tracks ECN (Explicit Congestion Notification) marks and congestion states per upstream host.
type Monitor struct {
	ecnCEMarks       atomic.Uint64
	totalPackets     atomic.Uint64
	window           *CongestionWindow
	currentCongested atomic.Uint32
}

// NewMonitor creates a new ECN congestion monitor.
func NewMonitor(windowCapacity int) *Monitor {
	return &Monitor{
		window: NewCongestionWindow(windowCapacity),
	}
}

// RecordPacket records packet reception with or without ECN Congestion Encountered (CE) bits.
func (m *Monitor) RecordPacket(ecnCE bool) {
	m.totalPackets.Add(1)
	if ecnCE {
		m.ecnCEMarks.Add(1)
		m.currentCongested.Store(1)
		m.window.Push(0) // Zero or reduced pacing window on congestion
	} else {
		m.currentCongested.Store(0)
		m.window.Push(100) // Nominal congestion window capacity
	}
}

// IsCongested returns true if recent packets indicate network congestion via ECN CE marks.
func (m *Monitor) IsCongested() bool {
	return m.currentCongested.Load() == 1
}

// Stats returns ECN packet statistics.
func (m *Monitor) Stats() (total uint64, ceMarks uint64) {
	return m.totalPackets.Load(), m.ecnCEMarks.Load()
}

// Window returns the underlying congestion window tracker.
func (m *Monitor) Window() *CongestionWindow {
	return m.window
}
