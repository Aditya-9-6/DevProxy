package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)

	if rb.CurrentStatus() != StatusUnknown {
		Errorf(t, "expected initial status to be unknown")
	}

	rb.Push(CheckResult{Status: StatusHealthy, Timestamp: time.Now()})
	if rb.CurrentStatus() != StatusHealthy {
		Errorf(t, "expected status healthy")
	}

	rb.Push(CheckResult{Status: StatusUnhealthy, Timestamp: time.Now()})
	rb.Push(CheckResult{Status: StatusHealthy, Timestamp: time.Now()})
	rb.Push(CheckResult{Status: StatusUnhealthy, Timestamp: time.Now()})

	snapshot := rb.Snapshot()
	if len(snapshot) != 3 {
		Errorf(t, "expected snapshot length 3, got %d", len(snapshot))
	}
}

func TestOrchestratorLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.Interval = 50 * time.Millisecond
	cfg.Timeout = 100 * time.Millisecond

	o := NewOrchestrator(cfg)
	o.AddTarget(Target{
		ID:  "test-target",
		URL: server.URL,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	o.Start(ctx)

	// Wait briefly for initial check
	time.Sleep(100 * time.Millisecond)

	status := o.GetStatus("test-target")
	if status != StatusHealthy {
		Errorf(t, "expected target to be healthy, got %s", status)
	}

	o.Stop()
}

func Errorf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	t.Errorf(format, args...)
}
