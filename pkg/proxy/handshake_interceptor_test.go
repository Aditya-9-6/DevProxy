package proxy

import (
	"context"
	"errors"
	"net"
	"testing"
)

type mockConn struct {
	net.Conn
}

func TestHandshakeInterceptor(t *testing.T) {
	interceptor := NewHandshakeInterceptor(80)

	ctx := context.Background()
	mc := &mockConn{}

	// Low entropy / botnet pattern
	maliciousHello := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01}
	err := interceptor.Intercept(ctx, mc, maliciousHello)
	if !errors.Is(err, ErrBotnetDetected) {
		t.Fatalf("expected ErrBotnetDetected, got %v", err)
	}

	// Normal high entropy pattern
	normalHello := []byte{0x16, 0x03, 0x01, 0x00, 0x20, 0x01, 0x0a, 0x3b, 0xff, 0x12, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}
	err = interceptor.Intercept(ctx, mc, normalHello)
	if err != nil {
		t.Fatalf("expected no error for normal hello, got %v", err)
	}
}
