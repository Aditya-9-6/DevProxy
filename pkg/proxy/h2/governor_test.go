package h2

import (
	"context"
	"sync"
	"testing"
)

func TestGovernor_FlowControl(t *testing.T) {
	g := NewGovernor(100)
	ctx := context.Background()
	streamID := uint32(1)
	g.RegisterStream(streamID, 50)

	t.Run("Successful Consumption", func(t *testing.T) {
		if err := g.RequestCredit(ctx, streamID, 20); err != nil {
			t.Errorf("Expected success, got %v", err)
		}
	})

	t.Run("Exhaustion", func(t *testing.T) {
		if err := g.RequestCredit(ctx, streamID, 100); err == nil {
			t.Error("Expected error due to exhaustion")
		}
	})

	t.Run("Partial Failure Restoration", func(t *testing.T) {
		// Stream has 30 left, Conn has 80 left. Request 40.
		// Stream should fail, Conn should not be deducted.
		initialConn := g.connWindow.Get()
		err := g.RequestCredit(ctx, streamID, 40)
		if err == nil {
			t.Error("Expected stream exhaustion error")
		}
		if g.connWindow.Get() != initialConn {
			t.Errorf("Connection window was incorrectly deducted: got %d, want %d", g.connWindow.Get(), initialConn)
		}
	})

	t.Run("Concurrent Access", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = g.RequestCredit(ctx, streamID, 1)
			}()
		}
		wg.Wait()
	})
}
