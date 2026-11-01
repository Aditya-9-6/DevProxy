package http2flow

import (
	"sync"
)

// Lease manages memory buffers and window allocation leases with zero allocations via sync.Pool.
type Lease struct {
	Data []byte
	Size int
}

var leasePool = sync.Pool{
	New: func() interface{} {
		return &Lease{
			Data: make([]byte, 16384), // Default HTTP/2 max frame payload size
		}
	},
}

// AcquireLease retrieves a Lease from the sync.Pool.
func AcquireLease() *Lease {
	l := leasePool.Get().(*Lease)
	l.Size = 0
	return l
}

// Release returns the Lease back to the sync.Pool.
func (l *Lease) Release() {
	if l == nil || cap(l.Data) < 16384 {
		return
	}
	l.Size = 0
	leasePool.Put(l)
}
