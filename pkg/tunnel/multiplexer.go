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

// Tunnel streams data between two connections, using splice if available.
func (m *Multiplexer) Tunnel(ctx context.Context, client, upstream *net.TCPConn) error {
	m.tracker.Increment()
	defer m.tracker.Decrement()

	errChan := make(chan error, 2)
	go func() {
		_, err := io.Copy(upstream, client)
		errChan <- err
	}()
	go func() {
		_, err := io.Copy(client, upstream)
		errChan <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
