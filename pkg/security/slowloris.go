package security

import (
	"net"
	"unsafe"
)

type SlowlorisFilter struct {
	sketch *CountMinSketch
	limit  uint64
}

func NewSlowlorisFilter(limit uint64) *SlowlorisFilter {
	return &SlowlorisFilter{
		sketch: NewCountMinSketch(),
		limit:  limit,
	}
}

func (f *SlowlorisFilter) Close() {
	if f.sketch != nil {
		f.sketch.Close()
	}
}

func (f *SlowlorisFilter) ShouldBlock(conn net.Conn) bool {
	addr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}

	// Zero-allocation string-to-byte conversion for hot path
	key := unsafeSlice(addr)

	f.sketch.Increment(key)
	return f.sketch.Estimate(key) > f.limit
}

func unsafeSlice(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
