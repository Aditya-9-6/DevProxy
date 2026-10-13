package tunnel

import (
	"context"
	"io"
	"net"
	"sync"
)

// Multiplexer manages bidirectional TCP tunneling.
type Multiplexer struct {
	tracker *ConnectionTracker
	pool    sync.Pool
}

func NewMultiplexer() *Multiplexer {
	return &Multiplexer{
		tracker: &ConnectionTracker{},
		pool: sync.Pool{
			New: func() interface{} { return make([]byte, 32*1024) },
		},
	}
}

// Tunnel streams data between two connections using pooled buffers and handles cancellation cleanly.
func (m *Multiplexer) Tunnel(ctx context.Context, client, upstream *net.TCPConn) error {
	m.tracker.Increment()
	defer m.tracker.Decrement()
	defer client.Close()
	defer upstream.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	copyStream := func(dst, src *net.TCPConn) {
		written, err := SpliceCopy(dst, src)
		if err == nil && written > 0 {
			errChan <- nil
			return
		}
		buf := m.pool.Get().([]byte)
		defer m.pool.Put(buf)
		_, err = io.CopyBuffer(dst, src, buf)
		errChan <- err
		cancel()
	}

	go copyStream(upstream, client)
	go copyStream(client, upstream)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
