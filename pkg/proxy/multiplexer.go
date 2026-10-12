package proxy

import (
	"net"
	"sync"
	"sync/atomic"
)

// Multiplexer manages a pool of upstream connections for HTTP/2 stream reuse.
type Multiplexer struct {
	conns sync.Map
	count atomic.Uint64
}

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{}
}

func (m *Multiplexer) Get(addr string) (net.Conn, bool) {
	if val, ok := m.conns.Load(addr); ok {
		return val.(net.Conn), true
	}
	return nil, false
}

func (m *Multiplexer) Put(addr string, conn net.Conn) {
	m.conns.Store(addr, conn)
	m.count.Add(1)
}

func (m *Multiplexer) Remove(addr string) {
	m.conns.Delete(addr)
	m.count.Add(^uint64(0))
}
