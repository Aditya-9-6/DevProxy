package proxy

import (
	"context"
	"fmt"
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

	c.Set(host, ips, 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	ipsRet, isStale, ok := c.Get(host)
	if !ok || !isStale {
		t.Error("expected expired DNS cache entry to be returned as stale for background revalidation")
	}
	if len(ipsRet) != 1 || ipsRet[0] != "127.0.0.1" {
		t.Error("expected original IPs in stale cache result")
	}
}

func TestHappyEyeballsDialer_RaceCondition(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	d := NewHappyEyeballsDialer()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	portStr := fmt.Sprintf("%d", listener.Addr().(*net.TCPAddr).Port)

	// Simulate multiple IPs where one works
	conn, err := d.raceDials(ctx, "tcp", []string{"127.0.0.1", "127.0.0.1"}, portStr)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	conn.Close()
}
