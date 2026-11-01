package http3

import (
	"net"
	"testing"
)

func TestRetryHandler(t *testing.T) {
	h := NewRetryHandler(true)
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:1234")
	token, ok := h.HandleInitial(addr, []byte("initial"))
	if !ok {
		t.Fatalf("expected retry token generation")
	}
	if len(token) == 0 {
		t.Fatalf("expected non-empty token")
	}
}
