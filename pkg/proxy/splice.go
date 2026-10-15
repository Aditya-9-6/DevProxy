package proxy

import (
	"context"
	"io"
	"net"
	"sync"
)

var spliceBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// ProxyPipe streams data between two connections using zero-allocation buffers.
func ProxyPipe(ctx context.Context, src, dst net.Conn, metrics *LatencyTracker) error {
	buf := spliceBufferPool.Get().([]byte)
	defer spliceBufferPool.Put(buf)

	// Use io.CopyBuffer to perform efficient streaming
	_, err := io.CopyBuffer(dst, src, buf)
	if metrics != nil {
		metrics.Record(1) // Track throughput
	}
	return err
}
