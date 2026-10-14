package ebpf

import (
	"context"
	"sync/atomic"
	"time"
)

// Poller manages the lifecycle of the eBPF ring buffer consumer.
type Poller struct {
	ctx       context.Context
	cancel    context.CancelFunc
	Events    chan []byte
	Dropped   atomic.Uint64
	Processed atomic.Uint64
}

func NewPoller() *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		ctx:    ctx,
		cancel: cancel,
		Events: make(chan []byte, 1024),
	}
}

func (p *Poller) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				// Poll eBPF ring buffer here
			}
		}
	}()
}

func (p *Poller) Stop() {
	p.cancel()
}
