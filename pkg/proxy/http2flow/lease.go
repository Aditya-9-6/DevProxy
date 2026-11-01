package http2flow

import (
	"errors"
)

var (
	// ErrWindowExhausted is returned when a flow control window is exceeded.
	ErrWindowExhausted = errors.New("http2flow: window exhausted")
)

// Lease represents a reservation of flow control capacity.
type Lease struct {
	ledger *Ledger
	streamID uint32
	size int32
}

// NewLease creates a flow control lease.
func NewLease(ledger *Ledger, streamID uint32, size int32) *Lease {
	return &Lease{
		ledger:   ledger,
		streamID: streamID,
		size:     size,
	}
}

// Size returns the lease size.
func (l *Lease) Size() int32 {
	return l.size
}
