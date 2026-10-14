package health

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	// Test failure
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts2.Close()

	orch.Probe(context.Background(), ts2.URL, ring)
	if ring.IsHealthy() {
		t.Error("Expected node to be unhealthy")
	}
}
