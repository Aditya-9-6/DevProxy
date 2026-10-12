package transport

import (
	"net"
	"sync"
)

// ConnectionWrapper provides a thin wrapper for tracking connection lifecycle.
type ConnectionWrapper struct {
	net.Conn
	mu     sync.Mutex
	closed bool
}

// Close ensures the connection is closed exactly once.
func (c *ConnectionWrapper) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.Conn.Close()
}
