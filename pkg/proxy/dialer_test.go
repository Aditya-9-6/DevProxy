package proxy

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/dns"
)

func TestHappyEyeballsDialer_ContextCancellation(t *testing.T) {
	d := NewHappyEyeballsDialer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := d.DialContext(ctx, "tcp", "google.com:80")
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}

func TestDNSCache_Eviction(t *testing.T) {
	c := dns.NewDNSCache()
	defer c.Close()

	host := "example.internal"
	ips := []string{"127.0.0.1"}

	// Set with very short TTL
	c.Set(host, ips, 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get(host)
	if ok {
		t.Error("expected expired DNS cache entry to be evicted/unavailable")
	}
}

func TestHappyEyeballsDialer_SuccessfulConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
	}()

	d := NewHappyEyeballsDialer()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("expected successful dial, got %v", err)
	}
	defer conn.Close()
}
