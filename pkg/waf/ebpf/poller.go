package ebpf

import (
	"context"
	"sync/atomic"
)

// Poller manages the lifecycle of the eBPF ring buffer consumer.
type Poller struct {
	ctx       context.Context
	cancel    context.CancelFunc
	Events    chan []byte
	Telemetry *Telemetry
	Dropped   atomic.Uint64
	Processed atomic.Uint64
}

func NewPoller() *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		ctx:       ctx,
		cancel:    cancel,
		Events:    make(chan []byte, 1024),
		Telemetry: &Telemetry{},
	}
}

func (p *Poller) Start() {
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			default:
				// Simulate eBPF ring buffer poll
				data := EventPool.Get().([]byte)
				// Simulate successful read
				select {
				case p.Events <- data:
					p.Processed.Add(1)
					p.Telemetry.RecordHit()
				default:
					// Backpressure: drop event if channel is full
					EventPool.Put(data)
					p.Dropped.Add(1)
					p.Telemetry.RecordDrop()
				}
			}
		}
	}()
}

func (p *Poller) Stop() {
	p.cancel()
}
