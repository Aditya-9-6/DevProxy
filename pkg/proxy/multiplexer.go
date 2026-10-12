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
	conns  sync.Map
	count  atomic.Uint64
	TTL    time.Duration
	ctx    context.Context
	cancel context.CancelFunc
}

func NewMultiplexer() *Multiplexer {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Multiplexer{
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
			m.conns.Range(func(key, value interface{}) bool {
				e := value.(entry)
				if time.Since(e.createdAt) > m.TTL {
					m.Remove(key.(string))
				}
				return true
			})
		}
	}
}

func (m *Multiplexer) Get(addr string) (net.Conn, bool) {
	val, ok := m.conns.Load(addr)
	if !ok {
		return nil, false
	}
	e := val.(entry)
	if time.Since(e.createdAt) > m.TTL {
		go m.Remove(addr)
		return nil, false
	}
	return e.conn, true
}

func (m *Multiplexer) Put(addr string, conn net.Conn) {
	m.conns.Store(addr, entry{conn: conn, createdAt: time.Now()})
	m.count.Add(1)
}

func (m *Multiplexer) Remove(addr string) {
	if val, ok := m.conns.LoadAndDelete(addr); ok {
		m.count.Add(^uint64(0))
		e := val.(entry)
		e.conn.Close()
	}
}

func (m *Multiplexer) Close() {
	m.cancel()
}
