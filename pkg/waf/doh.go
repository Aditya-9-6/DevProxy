package waf

import (
	"sync"
	"sync/atomic"
)

// DoHMetrics tracks DoH traffic statistics using lock-free atomic counters.
type DoHMetrics struct {
	MalformedQueries CacheLinePad
	Amplification    CacheLinePad
	RecordTypes      [256]atomic.Uint64
}

var GlobalDoHMetrics = &DoHMetrics{}

var dohBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

func (m *DoHMetrics) RecordMalformed() {
	atomic.AddUint64(&m.MalformedQueries[0], 1)
}

func (m *DoHMetrics) RecordAmplification() {
	atomic.AddUint64(&m.Amplification[0], 1)
}

func (m *DoHMetrics) RecordRecordType(t uint16) {
	if t < 256 {
		m.RecordTypes[t].Add(1)
	}
}
