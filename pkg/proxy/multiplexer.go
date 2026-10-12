package proxy

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type entry struct {
	conn      net.Conn
	createdAt time.Time
}

// Multiplexer manages a pool of upstream connections for HTTP/2 stream reuse.
type Multiplexer struct {
	mu     sync.RWMutex
	conns  map[string]entry
	count  atomic.Uint64
	TTL    time.Duration
	ctx    context.Context
	cancel context.CancelFunc
}

func NewMultiplexer() *Multiplexer {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Multiplexer{
		conns:  make(map[string]entry),
		TTL:    30 * time.Second,
		ctx:    ctx,
		cancel: cancel,
	}
	go m.reaper()
	return m
}

func (m *Multiplexer) reaper() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			for addr, e := range m.conns {
				if time.Since(e.createdAt) > m.TTL {
					delete(m.conns, addr)
					m.count.Add(^uint64(0))
					e.conn.Close()
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Multiplexer) Get(addr string) (net.Conn, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.conns[addr]
	if !ok {
		return nil, false
	}
	if time.Since(e.createdAt) > m.TTL {
		delete(m.conns, addr)
		m.count.Add(^uint64(0))
		e.conn.Close()
		return nil, false
	}
	return e.conn, true
}

func (m *Multiplexer) Put(addr string, conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.conns[addr]; !ok {
		m.count.Add(1)
	}
	m.conns[addr] = entry{conn: conn, createdAt: time.Now()}
}

func (m *Multiplexer) Remove(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.conns[addr]; ok {
		delete(m.conns, addr)
		m.count.Add(^uint64(0))
		e.conn.Close()
	}
}

func (m *Multiplexer) Close() {
	m.cancel()
}
