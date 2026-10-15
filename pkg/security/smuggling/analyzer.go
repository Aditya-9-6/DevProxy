package smuggling

import (
	"sync"
)

type Analyzer struct {
	pool sync.Pool
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		pool: sync.Pool{
			New: func() interface{} {
				return &ConnectionState{}
			},
		},
	}
}

func (a *Analyzer) GetState() *ConnectionState {
	return a.pool.Get().(*ConnectionState)
}

func (a *Analyzer) PutState(s *ConnectionState) {
	s.Reset()
	a.pool.Put(s)
}
