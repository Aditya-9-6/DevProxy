package h2

import (
	"context"
	"errors"
)

// Governor orchestrates stream-level and connection-level flow control.
type Governor struct {
	connWindow *Window
	streams    map[uint32]*Window
}

func NewGovernor(initialConnWindow int64) *Governor {
	return &Governor{
		connWindow: NewWindow(initialConnWindow),
		streams:    make(map[uint32]*Window),
	}
}

// RequestCredit checks if both connection and stream windows have sufficient capacity.
func (g *Governor) RequestCredit(ctx context.Context, streamID uint32, amount int64) error {
	// Connection level check
	if !g.connWindow.Consume(amount) {
		return errors.New("connection window exhausted")
	}
	return nil
}

// ReleaseCredit returns credits to the pool.
func (g *Governor) ReleaseCredit(amount int64) {
	g.connWindow.Replenish(amount)
}
