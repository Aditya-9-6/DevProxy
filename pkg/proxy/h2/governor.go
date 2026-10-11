package h2

import (
	"context"
	"fmt"
	"sync"
)

// Governor orchestrates stream-level and connection-level flow control.
type Governor struct {
	connWindow *Window
	mu         sync.RWMutex
	streams    map[uint32]*Window
	initial    int64
}

func NewGovernor(initialConnWindow int64) *Governor {
	return &Governor{
		connWindow: NewWindow(initialConnWindow),
		streams:    make(map[uint32]*Window),
		initial:    initialConnWindow,
	}
}

// RegisterStream initializes a window for a new stream.
func (g *Governor) RegisterStream(streamID uint32, initialWindow int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.streams[streamID] = NewWindow(initialWindow)
}

// UnregisterStream cleans up stream resources.
func (g *Governor) UnregisterStream(streamID uint32) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.streams, streamID)
}

// RequestCredit checks if both connection and stream windows have sufficient capacity.
func (g *Governor) RequestCredit(ctx context.Context, streamID uint32, amount int64) error {
	g.mu.RLock()
	stream, ok := g.streams[streamID]
	g.mu.RUnlock()

	if ok && !stream.Consume(amount) {
		return fmt.Errorf("stream window exhausted: %w", ErrFlowControlExhausted)
	}

	if !g.connWindow.Consume(amount) {
		if ok {
			stream.Replenish(amount)
		}
		return fmt.Errorf("connection window exhausted: %w", ErrFlowControlExhausted)
	}

	return nil
}

// ReleaseCredit returns credits to the connection pool.
func (g *Governor) ReleaseCredit(amount int64) {
	g.connWindow.Replenish(amount)
}

var ErrFlowControlExhausted = fmt.Errorf("flow control credit exhausted")
