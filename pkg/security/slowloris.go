package security

import (
	"net"
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

func (f *SlowlorisFilter) ShouldBlock(conn net.Conn) bool {
	addr, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return false
	}
	key := []byte(addr)
	f.sketch.Increment(key)
	return f.sketch.Estimate(key) > f.limit
}
