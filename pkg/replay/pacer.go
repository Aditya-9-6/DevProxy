package replay

import (
	"sync/atomic"
	"time"
)

type Pacer struct {
	lastEventTime atomic.Int64
}

func NewPacer() *Pacer {
	return &Pacer{}
}

func (p *Pacer) Wait(eventTime time.Time) {
	last := p.lastEventTime.Swap(eventTime.UnixNano())
	if last == 0 {
		return
	}
	diff := eventTime.Sub(time.Unix(0, last))
	if diff > 0 {
		time.Sleep(diff)
	}
}
