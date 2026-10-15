package tls

import (
	"sync/atomic"
	"time"
)

// OCSPStaple represents a cached OCSP response.
type OCSPStaple struct {
	Response []byte
	Expiry   time.Time
}

// CacheLinePad prevents false sharing by padding structures to 64 bytes.
type CacheLinePad [8]uint64

// StapleEntry is a cache-aligned container for an atomic OCSP staple pointer.
type StapleEntry struct {
	_      CacheLinePad
	Staple atomic.Pointer[OCSPStaple]
	_      CacheLinePad
}
