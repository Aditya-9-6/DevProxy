package proxy

import (
	"net"
	"testing"
)

func TestMultiplexer(t *testing.T) {
	m := NewMultiplexer()
	addr := "127.0.0.1:8080"

	// Mock connection
	conn, _ := net.Pipe()
	m.Put(addr, conn)

	got, ok := m.Get(addr)
	if !ok || got != conn {
		t.Errorf("failed to retrieve connection")
	}

	m.Remove(addr)
	_, ok = m.Get(addr)
	if ok {
		t.Errorf("connection should have been removed")
	}
}
