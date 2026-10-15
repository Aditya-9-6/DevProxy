package replay

import (
	"context"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"sync"
)

var contextPool = sync.Pool{
	New: func() interface{} {
		return &ReplayContext{}
	},
}

type ReplayContext struct {
	Event *ringbuffer.TrafficEvent
}

type Harness struct {
	buffer *ringbuffer.RingBuffer
	pacer  *Pacer
}

func NewHarness(size int) *Harness {
	return &Harness{
		buffer: ringbuffer.NewRingBuffer(size),
		pacer:  NewPacer(),
	}
}

func (h *Harness) Replay(ctx context.Context, handler func(*ringbuffer.TrafficEvent)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			event := h.buffer.TryPop()
			if event == nil {
				return
			}
			h.pacer.Wait(event.Timestamp)
			handler(event)
		}
	}
}
