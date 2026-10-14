package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
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
}

func TestOrchestrator_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ring := &StateRing{}
	orch := NewOrchestrator()
	orch.Probe(ctx, "http://localhost:1", ring)
	if ring.IsHealthy() {
		t.Error("Expected unhealthy due to cancelled context")
	}
}
