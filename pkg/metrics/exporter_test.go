package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrometheusExporter(t *testing.T) {
	exp := GetExporter()

	// Record some test metrics
	exp.IncActiveConns()
	exp.AddBytesSent(1024)
	exp.AddBytesReceived(2048)
	exp.IncRingBufferDrops()
	exp.RecordLatency(50 * time.Millisecond)
	exp.RecordLatency(120 * time.Millisecond)

	handler := exp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(bodyBytes)

	expectedMetrics := []string{
		"devproxy_active_connections",
		"devproxy_bytes_sent_total",
		"devproxy_bytes_received_total",
		"devproxy_ringbuffer_drops_total",
		"devproxy_goroutines",
		"devproxy_latency_p50_ms",
		"devproxy_latency_p95_ms",
		"devproxy_latency_p99_ms",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(bodyStr, m) {
			t.Errorf("expected metrics output to contain %q, got:\n%s", m, bodyStr)
		}
	}

	// Cleanup / decrease active conns
	exp.DecActiveConns()
}
