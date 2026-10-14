package health

import (
	"context"
	"net/http"
	"time"
)

// Orchestrator manages health probes for upstream targets.
type Orchestrator struct {
	client *http.Client
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (o *Orchestrator) Probe(ctx context.Context, url string, ring *StateRing) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		ring.SetHealthy(false)
		return
	}

	resp, err := o.client.Do(req)
	if err != nil {
		ring.SetHealthy(false)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		ring.SetHealthy(false)
		return
	}

	ring.SetHealthy(true)
	ring.RecordLatency(uint64(time.Since(start).Milliseconds()))
}
