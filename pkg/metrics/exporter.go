package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// PrometheusExporter collects and formats real-time proxy metrics in Prometheus text exposition format.
type PrometheusExporter struct {
	activeConns     atomic.Int64
	bytesSent       atomic.Uint64
	bytesReceived   atomic.Uint64
	ringBufferDrops atomic.Uint64
	latencies       [1000]atomic.Uint64 // Fixed lock-free atomic latency buckets (milliseconds)
	mu              sync.RWMutex
}

var (
	globalExporter *PrometheusExporter
	once           sync.Once
)

// GetExporter returns the singleton PrometheusExporter instance.
func GetExporter() *PrometheusExporter {
	once.Do(func() {
		globalExporter = &PrometheusExporter{}
	})
	return globalExporter
}

// IncActiveConns increments active connection count.
func (e *PrometheusExporter) IncActiveConns() {
	e.activeConns.Add(1)
}

// DecActiveConns decrements active connection count.
func (e *PrometheusExporter) DecActiveConns() {
	e.activeConns.Add(-1)
}

// AddBytesSent records outbound bytes.
func (e *PrometheusExporter) AddBytesSent(n uint64) {
	e.bytesSent.Add(n)
}

// AddBytesReceived records inbound bytes.
func (e *PrometheusExporter) AddBytesReceived(n uint64) {
	e.bytesReceived.Add(n)
}

// IncRingBufferDrops increments ring buffer drop counter.
func (e *PrometheusExporter) IncRingBufferDrops() {
	e.ringBufferDrops.Add(1)
}

// RecordLatency records upstream latency in an O(1) lock-free atomic bucket.
func (e *PrometheusExporter) RecordLatency(d time.Duration) {
	ms := d.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	idx := int(ms)
	if idx >= len(e.latencies) {
		idx = len(e.latencies) - 1
	}
	e.latencies[idx].Add(1)
}

// Handler returns an http.Handler that serves Prometheus metrics.
func (e *PrometheusExporter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		var sb []byte
		// Estimate capacity to reduce re-allocations
		sb = make([]byte, 0, 1024)

		appendMetric := func(name, help, mtype string, val interface{}) {
			sb = append(sb, fmt.Sprintf("# HELP %s %s\n# TYPE %s %s\n%s %v\n", name, help, name, mtype, name, val)...)
		}

		appendMetric("devproxy_active_connections", "Current active proxy connections", "gauge", e.activeConns.Load())
		appendMetric("devproxy_bytes_sent_total", "Total bytes sent upstream/downstream", "counter", e.bytesSent.Load())
		appendMetric("devproxy_bytes_received_total", "Total bytes received", "counter", e.bytesReceived.Load())
		appendMetric("devproxy_ringbuffer_drops_total", "Total dropped events due to full ring buffer", "counter", e.ringBufferDrops.Load())
		appendMetric("devproxy_goroutines", "Current active goroutines", "gauge", runtime.NumGoroutine())

		// Latency percentiles calculation without heap slice allocation
		var totalCount uint64
		var counts [1000]uint64
		for i := 0; i < 1000; i++ {
			c := e.latencies[i].Load()
			counts[i] = c
			totalCount += c
		}

		var p50, p95, p99 float64
		if totalCount > 0 {
			target50 := totalCount / 2
			target95 := (totalCount * 95) / 100
			target99 := (totalCount * 99) / 100

			var running uint64
			found50, found95, found99 := false, false, false

			for i := 0; i < 1000; i++ {
				running += counts[i]
				if !found50 && running >= target50 {
					p50 = float64(i)
					found50 = true
				}
				if !found95 && running >= target95 {
					p95 = float64(i)
					found95 = true
				}
				if !found99 && running >= target99 {
					p99 = float64(i)
					found99 = true
				}
			}
		}

		appendMetric("devproxy_latency_p50_ms", "Upstream response latency 50th percentile in milliseconds", "gauge", p50)
		appendMetric("devproxy_latency_p95_ms", "Upstream response latency 95th percentile in milliseconds", "gauge", p95)
		appendMetric("devproxy_latency_p99_ms", "Upstream response latency 99th percentile in milliseconds", "gauge", p99)

		_, _ = w.Write(sb)
	})
}
