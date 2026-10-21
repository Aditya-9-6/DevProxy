package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthOrchestrator_AddRemoveNode(t *testing.T) {
	orch := NewHealthOrchestrator(3, 2, 100*time.Millisecond)
	defer orch.Stop()

	orch.AddNode("node1", "http://example.com/health")
	if _, ok := orch.Status("node1"); !ok {
		t.Error("Expected node1 to exist")
	}

	orch.RemoveNode("node1")
	if _, ok := orch.Status("node1"); ok {
		t.Error("Expected node1 to be removed")
	}
}

func TestHealthOrchestrator_StatusTransition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(2, 2, 50*time.Millisecond)
	defer orch.Stop()

	orch.AddNode("node1", server.URL)

	time.Sleep(500 * time.Millisecond)

	status, ok := orch.Status("node1")
	if !ok {
		t.Fatal("Node not found")
	}
	if status != 1 {
		t.Errorf("Expected status 1 (healthy), got %d", status)
	}
}

func TestHealthOrchestrator_StopCancel(t *testing.T) {
	orch := NewHealthOrchestrator(3, 2, 100*time.Millisecond)

	orch.AddNode("node1", "http://example.com/health")
	orch.AddNode("node2", "http://example.org/health")

	orch.Stop()

	time.Sleep(100 * time.Millisecond)

	if orch.ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}
