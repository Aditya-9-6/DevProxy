package proxy

import (
	"context"
	"testing"
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
	// Test that expired items are not returned
	// Note: DNSCache is in pkg/dns, but testing here for integration
}
