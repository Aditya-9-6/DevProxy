package quic

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Aditya-9-6/DevProxy/pkg/memory/arena"
)

// Multiplexer handles QUIC datagrams with lock-free handoff.
type Multiplexer struct {
	allocator *arena.SlabAllocator
	conns     sync.Map
	active    atomic.Uint64
}

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		allocator: arena.NewSlabAllocator(1500), // Standard MTU
	}
}

func (m *Multiplexer) Dispatch(ctx context.Context, connID string, data []byte) error {
	buf := m.allocator.Get()
	n := copy(buf, data)

	// Lock-free handoff simulation
	m.active.Add(1)
	defer m.active.Add(^uint64(0))

	// Logic to route to specific stream/connection
	_ = connID
	_ = buf[:n]

	m.allocator.Put(buf)
	return nil
}
