package analysis

import (
	"sync/atomic"
	"github.com/Aditya-9-6/DevProxy/pkg/http2/goaway"
)

// Observer tracks aggregate phase and error counters for GOAWAY events without retaining debug data.
type Observer struct {
	goawayCount   uint64
	drainingCount uint64
	errorCount    uint64
}

func NewObserver() *Observer {
	return &Observer{}
}

func (o *Observer) Observe(f *goaway.Frame, err error) {
	if err != nil {
		atomic.AddUint64(&o.errorCount, 1)
		return
	}
	if f != nil {
		atomic.AddUint64(&o.goawayCount, 1)
	}
}

func (o *Observer) Counts() (goaways uint64, errors uint64) {
	return atomic.LoadUint64(&o.goawayCount), atomic.LoadUint64(&o.errorCount)
}
