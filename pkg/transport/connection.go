package transport

import (
	"net"
	"sync"

	"github.com/Aditya-9-6/DevProxy/pkg/proxy/drain"
)

// ConnectionWrapper provides a thin wrapper for tracking connection lifecycle and drain integration.
type ConnectionWrapper struct {
	net.Conn
	manager *drain.Manager
	state   *drain.ConnectionState
	mu      sync.Mutex
	closed  bool
}

// NewConnectionWrapper wraps a net.Conn with drain manager tracking.
func NewConnectionWrapper(conn net.Conn, manager *drain.Manager) *ConnectionWrapper {
	var state *drain.ConnectionState
	if manager != nil {
		if manager.Register() {
			state = manager.GetState()
		}
	}
	return &ConnectionWrapper{
		Conn:    conn,
		manager: manager,
		state:   state,
	}
}

// Close ensures the connection is closed exactly once and unregisters from the drain manager.
func (c *ConnectionWrapper) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true

	if c.manager != nil {
		if c.state != nil {
			c.manager.PutState(c.state)
			c.state = nil
		}
		c.manager.Unregister()
	}

	return c.Conn.Close()
}
