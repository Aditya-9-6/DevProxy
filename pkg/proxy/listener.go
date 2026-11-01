package proxy

import (
	"context"
	"net"
	"sync"
)

type Listener struct {
	net.Listener
	mu     sync.Mutex
	closed bool
}

func NewListener(l net.Listener) *Listener {
	return &Listener{Listener: l}
}

func (l *Listener) AcceptConn(ctx context.Context) (net.Conn, error) {
	connChan := make(chan struct {
		conn net.Conn
		err  error
	}, 1)

	go func() {
		conn, err := l.Accept()
		connChan <- struct {
			conn net.Conn
			err  error
		}{conn, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-connChan:
		return res.conn, res.err
	}
}
