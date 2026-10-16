package smuggling

import "sync/atomic"

const (
	FlagContentLength uint64 = 1 << iota
	FlagTransferEncoding
	FlagChunked
)

type ConnectionState struct {
	HeaderMask atomic.Uint64
}

func (s *ConnectionState) Reset() {
	s.HeaderMask.Store(0)
}

func (s *ConnectionState) ObserveHeader(key string, value string) {
	switch key {
	case "Content-Length":
		s.HeaderMask.Or(FlagContentLength)
	case "Transfer-Encoding":
		s.HeaderMask.Or(FlagTransferEncoding)
		if value == "chunked" {
			s.HeaderMask.Or(FlagChunked)
		}
	}
}

func (s *ConnectionState) IsSmugglingRisk() bool {
	mask := s.HeaderMask.Load()
	return (mask&FlagContentLength) != 0 && (mask&FlagTransferEncoding) != 0
}
