package proxy

import (
	"context"
	"time"
)

type Watchdog struct {
	wheel *TimingWheel
}

func NewWatchdog() *Watchdog {
	return &Watchdog{wheel: NewTimingWheel()}
}

func (w *Watchdog) Start(ctx context.Context) {
	ticker := time.NewTicker(TickMs * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.wheel.Tick()
			_ = w.wheel.GetExpiredCount()
		}
	}
}

func (w *Watchdog) Track(deadline time.Duration) {
	w.wheel.Register(deadline)
}
