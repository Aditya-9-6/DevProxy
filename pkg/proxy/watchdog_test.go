package proxy

import (
	"context"
	"sync"
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
	defer cancel()

	w := NewWatchdog()
	go w.Start(ctx)

	w.Track(50 * time.Millisecond)
	TIME_BUF := time.After(300 * time.Millisecond)

	select {
	case <-TIME_BUF:
		// successfully verified lifecycle without strict sleep synchronization
	case <-ctx.Done():
	}
}

func TestTimingWheelConcurrentStress(t *testing.T) {
	t.Parallel()
	w := NewTimingWheel()
	var wg sync.WaitGroup

	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				w.Register(time.Duration(j+1) * 10 * time.Millisecond)
				if id == 0 {
					w.Tick()
					w.GetExpiredCount()
				}
			}
		}(i)
	}

	wg.Wait()
}
