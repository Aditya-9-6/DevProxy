package qpack

// Limits defines strict bounds for QPACK parsing and decoding to prevent resource exhaustion.
type Limits struct {
	MaxHeaderListSize      uint64
	MaxTableCapacity       uint64
	MaxBlockedStreams      int
	MaxHeaderFieldLength   int
}

// DefaultLimits returns sensible secure defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxHeaderListSize:    64 * 1024,
		MaxTableCapacity:     4096,
		MaxBlockedStreams:    16,
		MaxHeaderFieldLength: 8192,
	}
}
