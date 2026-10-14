package health

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

// Orchestrator manages health probes for upstream targets.
type Orchestrator struct {
	client *http.Client
}

func NewOrchestrator() *Orchestrator {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &Orchestrator{
		client: &http.Client{
			Timeout:   2 * time.Second,
			Transport: transport,
		},
	}
}

// StartBackgroundProbing runs a ticker loop to probe the target periodically.
func (o *Orchestrator) StartBackgroundProbing(ctx context.Context, url string, ring *StateRing, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.Probe(ctx, url, ring)
			}
		}
	}()
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

	// Limit response body to 1KB to prevent resource exhaustion
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 500 {
		ring.SetHealthy(false)
		return
	}

	ring.SetHealthy(true)
	ring.RecordLatency(uint64(time.Since(start).Milliseconds()))
}
