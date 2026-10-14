package proxy

import (
	"context"
	"testing"
	"time"
)

func TestTimingWheel(t *testing.T) {
	w := NewTimingWheel()
	w.Register(200 * time.Millisecond)

	w.Tick()
	w.Tick()

	if count := w.GetExpiredCount(); count != 1 {
		t.Errorf("expected 1 expired, got %d", count)
	}
}

func TestWatchdogLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := NewWatchdog()
	go w.Start(ctx)
	w.Track(100 * time.Millisecond)
	time.Sleep(250 * time.Millisecond)
	cancel()
}
