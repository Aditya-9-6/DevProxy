package tcp

import "sync/atomic"

type Metrics struct {
	TotalBytes atomic.Uint64
	DropCount  atomic.Uint64
}

func (m *Metrics) RecordTransfer(n uint64) {
	m.TotalBytes.Add(n)
}

func (m *Metrics) RecordDrop() {
	m.DropCount.Add(1)
}
