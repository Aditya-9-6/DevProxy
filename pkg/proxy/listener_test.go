package proxy

import (
	"context"
	"net"
	"testing"
)

type mockListener struct {
	net.Listener
	connChan chan net.Conn
}

func (m *mockListener) Accept() (net.Conn, error) {
	conn, ok := <-m.connChan
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (m *mockListener) Close() error {
	close(m.connChan)
	return nil
}

func TestListenerAccept(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	ml := &mockListener{
		connChan: make(chan net.Conn, 1),
	}
	ml.connChan <- c1

	l := NewListener(ml)
	defer l.Close()

	ctx := context.Background()
	conn, err := l.AcceptConn(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatalf("expected connection, got nil")
	}
	conn.Close()
}
