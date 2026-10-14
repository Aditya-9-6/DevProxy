package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestOrchestrator_Probe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ring := &StateRing{}
	orch := NewOrchestrator()

	orch.Probe(context.Background(), ts.URL, ring)

	if !ring.IsHealthy() {
		t.Error("Expected node to be healthy")
	}

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts2.Close()

	orch.Probe(context.Background(), ts2.URL, ring)
	if ring.IsHealthy() {
		t.Error("Expected node to be unhealthy")
	}
}

func TestOrchestrator_ConcurrentAccess(t *testing.T) {
	ring := &StateRing{}
	orch := NewOrchestrator()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			orch.Probe(context.Background(), ts.URL, ring)
		}()
	}
	wg.Wait()
	if !ring.IsHealthy() {
		t.Error("Expected healthy state after concurrent probes")
	}
}

func TestOrchestrator_LatencyBuckets(t *testing.T) {
	ring := &StateRing{}
	ring.RecordLatency(5)
	ring.RecordLatency(15)
	ring.RecordLatency(995)

	if ring.LatencyBuckets[0].Load() != 1 {
		t.Errorf("Expected bucket 0 to have 1, got %d", ring.LatencyBuckets[0].Load())
	}
	if ring.LatencyBuckets[1].Load() != 1 {
		t.Errorf("Expected bucket 1 to have 1, got %d", ring.LatencyBuckets[1].Load())
	}
	if ring.LatencyBuckets[99].Load() != 1 {
		t.Errorf("Expected bucket 99 to have 1, got %d", ring.LatencyBuckets[99].Load())
	}
}

func TestOrchestrator_BackgroundProbing(t *testing.T) {
	ring := &StateRing{}
	orch := NewOrchestrator()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	orch.StartBackgroundProbing(ctx, ts.URL, ring, 20*time.Millisecond)
	time.Sleep(60 * time.Millisecond)

	if !ring.IsHealthy() {
		t.Error("Expected healthy state from background probe")
	}
}
