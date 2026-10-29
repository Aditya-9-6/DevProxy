//go:build !linux

package splice

import (
	"context"
	"io"
	"net"
	"sync"
)

var splicePool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
		return &b
	},
}

// SpliceCopy copies data between two connections with zero heap allocation per call.
func SpliceCopy(ctx context.Context, dst net.Conn, src net.Conn) (int64, error) {
	bufPtr := splicePool.Get().(*[]byte)
	defer splicePool.Put(bufPtr)

	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		nr, err := src.Read(*bufPtr)
		if nr > 0 {
			nw, ew := dst.Write((*bufPtr)[:nr])
			if nw > 0 {
				total += int64(nw)
			}
			if ew != nil {
				return total, ew
			}
		}
		if err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
	}
}
