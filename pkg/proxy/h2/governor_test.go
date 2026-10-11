package h2

import (
	"context"
	"testing"
)

func TestGovernor_FlowControl(t *testing.T) {
	g := NewGovernor(100)
	ctx := context.Background()

	t.Run("Successful Consumption", func(t *testing.T) {
		if err := g.RequestCredit(ctx, 1, 50); err != nil {
			t.Errorf("Expected success, got %v", err)
		}
		if g.connWindow.Get() != 50 {
			t.Errorf("Expected 50 credits, got %d", g.connWindow.Get())
		}
	})

	t.Run("Exhaustion", func(t *testing.T) {
		if err := g.RequestCredit(ctx, 1, 60); err == nil {
			t.Error("Expected error due to exhaustion")
		}
	})

	t.Run("Replenishment", func(t *testing.T) {
		g.ReleaseCredit(50)
		if g.connWindow.Get() != 100 {
			t.Errorf("Expected 100 credits, got %d", g.connWindow.Get())
		}
	})
}
