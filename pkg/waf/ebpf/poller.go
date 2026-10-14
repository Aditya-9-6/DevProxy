package ebpf

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Poller manages the lifecycle of the eBPF ring buffer consumer.
type Poller struct {
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
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
	p.startOnce.Do(func() {
		go func() {
			for {
				select {
				case <-p.ctx.Done():
					return
				default:
					// Simulate blocking eBPF poll to avoid 100% CPU usage
					time.Sleep(1 * time.Millisecond)
					data := EventPool.Get().([]byte)
					select {
					case p.Events <- data:
						p.Processed.Add(1)
						p.Telemetry.RecordHit()
					default:
						EventPool.Put(data)
						p.Dropped.Add(1)
						p.Telemetry.RecordDrop()
					}
				}
			}
		}()
	})
}

func (p *Poller) Stop() {
	p.cancel()
}
