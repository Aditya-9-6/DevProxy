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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, 2)

	copyStream := func(dst, src io.Writer, reader io.Reader) {
		buf := m.pool.Get().([]byte)
		defer m.pool.Put(buf)
		_, err := io.CopyBuffer(dst, reader, buf)
		errChan <- err
		cancel()
	}

	go copyStream(upstream, client, client)
	go copyStream(client, upstream, upstream)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
