package waf

import (
	"sync/atomic"
)

// RevocationEpoch tracks the global revocation state using cache-line padded atomics.
// Uses explicit alignment to ensure the atomic field sits on its own cache line.
type RevocationEpoch struct {
	_       [56]byte
	Version atomic.Uint64
	_       [56]byte
}

func NewRevocationEpoch() *RevocationEpoch {
	return &RevocationEpoch{}
}

func (r *RevocationEpoch) RevokeAll() {
	r.Version.Add(1)
}

func (r *RevocationEpoch) GetCurrentEpoch() uint64 {
	return r.Version.Load()
}
