package reaper

import (
	"context"
	"net"
	"sync"
	"time"
)

// IdleConn tracks a connection and its last usage time.
type IdleConn struct {
	Conn     net.Conn
	LastUsed time.Time
	Hazard   *HazardPointer
}

// Reaper manages idle connection cleanup.
type Reaper struct {
	mu       sync.Mutex
	conns    map[string]*IdleConn
	interval time.Duration
}

func NewReaper(interval time.Duration) *Reaper {
	return &Reaper{
		conns:    make(map[string]*IdleConn),
		interval: interval,
	}
}

func (r *Reaper) Register(id string, conn net.Conn, hazard *HazardPointer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[id] = &IdleConn{Conn: conn, LastUsed: time.Now(), Hazard: hazard}
}

func (r *Reaper) Start(ctx context.Context, idleTimeout time.Duration) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reap(idleTimeout)
		}
	}
}

func (r *Reaper) reap(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, ic := range r.conns {
		if time.Since(ic.LastUsed) > timeout && !ic.Hazard.IsActive() {
			ic.Conn.Close()
			delete(r.conns, id)
		}
	}
}
