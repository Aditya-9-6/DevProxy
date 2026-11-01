package metrics

import (
	"sync/atomic"
	"github.com/Aditya-9-6/DevProxy/pkg/tls/certcomp"
)

// CertCompMetrics collects telemetry counters for TLS certificate compression.
type CertCompMetrics struct {
	totalHandshakes     atomic.Int64
	totalCompressed     atomic.Int64
	totalRejections     atomic.Int64
	totalDecompressBytes atomic.Int64
}

var GlobalCertCompMetrics = &CertCompMetrics{}

func (m *CertCompMetrics) RecordNegotiation(alg certcomp.Algorithm, success bool) {
	m.totalHandshakes.Add(1)
	if success {
		m.totalCompressed.Add(1)
	} else {
		m.totalRejections.Add(1)
	}
}

func (m *CertCompMetrics) RecordRejection() {
	m.totalRejections.Add(1)
}

func (m *CertCompMetrics) Snapshot() (handshakes, compressed, rejections int64) {
	return m.totalHandshakes.Load(), m.totalCompressed.Load(), m.totalRejections.Load()
}