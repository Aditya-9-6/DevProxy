package waf

import (
	"sync/atomic"
)

// RevocationEpoch tracks the global revocation state using cache-line padded atomics.
// This prevents false sharing during high-concurrency validation checks.
type RevocationEpoch struct {
	_       [56]byte // Padding for 64-byte cache line alignment
	Version atomic.Uint64
	_       [56]byte
}

// GlobalRevocationState is the singleton instance for the proxy.
var GlobalRevocationState = &RevocationEpoch{}

// RevokeAll increments the epoch, effectively invalidating all previous JWTs.
func (r *RevocationEpoch) RevokeAll() {
	r.Version.Add(1)
}

// GetCurrentEpoch returns the current revocation epoch.
func (r *RevocationEpoch) GetCurrentEpoch() uint64 {
	return r.Version.Load()
}
