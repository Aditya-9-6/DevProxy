package proxy

import (
	"context"
	"testing"
	"time"
)

func TestUpstreamTransport_Probes(t *testing.T) {
	transport := NewUpstreamTransport(false)
	transport.probeInterval = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go transport.StartProbes(ctx)

	select {
	case <-ctx.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Probes did not terminate within expected time")
	}
	transport.Close()
}
