package correlation

import (
	"sync"
)

// Pool manages a pre-allocated set of correlation objects.
type Pool struct {
	syncPool sync.Pool
}

func NewPool() *Pool {
	return &Pool{
		syncPool: sync.Pool{
			New: func() interface{} {
				return &CorrelationEntry{}
			},
		},
	}
}

func (p *Pool) Get() *CorrelationEntry {
	entry := p.syncPool.Get().(*CorrelationEntry)
	entry.Active.Store(true)
	return entry
}

func (p *Pool) Put(entry *CorrelationEntry) {
	p.syncPool.Put(entry)
}
