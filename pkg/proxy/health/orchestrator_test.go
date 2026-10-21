package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHealthOrchestrator_AddRemoveNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(1, 1, 5*time.Millisecond)
	defer orch.Stop()

	orch.AddNode("node1", server.URL)
	if _, ok := orch.Status("node1"); !ok {
		t.Error("Expected node1 to exist")
	}
	if ring, ok := orch.Ring("node1"); !ok || ring == nil {
		t.Error("Expected node1 to have a health ring")
	}

	orch.RemoveNode("node1")
	if _, ok := orch.Status("node1"); ok {
		t.Error("Expected node1 to be removed")
	}
}

func TestHealthOrchestrator_StatusTransition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(2, 2, 10*time.Millisecond)
	defer orch.Stop()

	orch.AddNode("node1", server.URL)
	waitForOrchestratorStatusForTest(t, orch, "node1", nodeHealthStatusHealthy)
}

func TestHealthOrchestrator_UnhealthyStatusTransition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(2, 2, 10*time.Millisecond)
	defer orch.Stop()

	orch.AddNode("node1", server.URL)
	waitForOrchestratorStatusForTest(t, orch, "node1", nodeHealthStatusUnhealthy)
}

func TestHealthOrchestrator_StopCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(3, 2, 10*time.Millisecond)
	orch.AddNode("node1", server.URL)

	orch.Stop()
	if err := orch.ctx.Err(); err != context.Canceled {
		t.Errorf("Expected context to be cancelled, got %v", err)
	}

	// Stop is idempotent.
	orch.Stop()

	// A stopped orchestrator must not start a new WaitGroup task.
	orch.AddNode("late", server.URL)
	if _, ok := orch.Status("late"); ok {
		t.Error("Expected a stopped orchestrator not to add nodes")
	}
}

func TestHealthOrchestrator_ConcurrentAddAndStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	orch := NewHealthOrchestrator(1, 1, 5*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			orch.AddNode("shared", server.URL)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		orch.Stop()
	}()

	wg.Wait()
	orch.Stop()
}

func TestNodeHealthChecker_Probe(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	checker := NewNodeHealthChecker(context.Background(), healthy.URL, healthy.Client(), 1, 1, time.Millisecond, nil)
	if err := checker.probe(); err != nil {
		t.Fatalf("Expected successful probe, got %v", err)
	}

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	failingChecker := NewNodeHealthChecker(context.Background(), unhealthy.URL, unhealthy.Client(), 1, 1, time.Millisecond, nil)
	if err := failingChecker.probe(); err == nil {
		t.Fatal("Expected non-2xx response to fail the probe")
	}

	invalidChecker := NewNodeHealthChecker(context.Background(), "http://[::1", http.DefaultClient, 1, 1, time.Millisecond, nil)
	if err := invalidChecker.probe(); err == nil {
		t.Fatal("Expected malformed URL to fail the probe")
	}

	defaultChecker := NewNodeHealthChecker(nil, healthy.URL, nil, 1, 1, time.Millisecond, nil)
	if err := defaultChecker.probe(); err != nil {
		t.Fatalf("Expected nil context and client to use safe defaults, got %v", err)
	}
}

func TestHealthDurationToUint64(t *testing.T) {
	maxDuration := time.Duration(1<<63 - 1)
	tests := []struct {
		name     string
		duration time.Duration
		want     uint64
	}{
		{name: "negative", duration: -time.Nanosecond, want: 0},
		{name: "zero", duration: 0, want: 0},
		{name: "positive", duration: time.Nanosecond, want: 1},
		{name: "maximum", duration: maxDuration, want: uint64(1<<63 - 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := healthDurationToUint64(test.duration); got != test.want {
				t.Errorf("healthDurationToUint64(%v) = %d, want %d", test.duration, got, test.want)
			}
		})
	}
}

func TestHealthCheckIntervalBounds(t *testing.T) {
	maxDuration := time.Duration(1<<63 - 1)
	increaseTests := []struct {
		name     string
		current  time.Duration
		maximum  time.Duration
		expected time.Duration
	}{
		{name: "double", current: 10, maximum: 80, expected: 20},
		{name: "exact maximum", current: 40, maximum: 80, expected: 80},
		{name: "cap before overflow", current: 50, maximum: 80, expected: 80},
		{name: "already maximum", current: 80, maximum: 80, expected: 80},
		{name: "maximum duration", current: maxDuration, maximum: maxDuration, expected: maxDuration},
	}

	for _, test := range increaseTests {
		t.Run("increase/"+test.name, func(t *testing.T) {
			if got := increaseHealthCheckInterval(test.current, test.maximum); got != test.expected {
				t.Errorf("increaseHealthCheckInterval(%v, %v) = %v, want %v", test.current, test.maximum, got, test.expected)
			}
		})
	}

	decreaseTests := []struct {
		name     string
		current  time.Duration
		minimum  time.Duration
		expected time.Duration
	}{
		{name: "halve", current: 20, minimum: 3, expected: 10},
		{name: "use minimum", current: 4, minimum: 3, expected: 3},
		{name: "below minimum", current: 2, minimum: 3, expected: 3},
		{name: "at minimum", current: 3, minimum: 3, expected: 3},
	}

	for _, test := range decreaseTests {
		t.Run("decrease/"+test.name, func(t *testing.T) {
			if got := decreaseHealthCheckInterval(test.current, test.minimum); got != test.expected {
				t.Errorf("decreaseHealthCheckInterval(%v, %v) = %v, want %v", test.current, test.minimum, got, test.expected)
			}
		})
	}
}

func waitForOrchestratorStatusForTest(t *testing.T, orch *HealthOrchestrator, id string, expected uint64) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		status, ok := orch.Status(id)
		if !ok {
			t.Fatalf("Node %q not found", id)
		}
		if status == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("Node %q status = %d, want %d", id, status, expected)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
