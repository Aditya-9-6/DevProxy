package proxy

import (
	"context"
	"testing"
	"time"
)

func TestUpstreamTransport_Probes(t *testing.T) {
	transport := NewUpstreamTransport(false)
	transport.probeInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go transport.StartProbes(ctx)

	time.Sleep(30 * time.Millisecond)
	transport.Close()
}

func TestParseUpstreamProxy(t *testing.T) {
	tests := []struct {
		raw     string
		wantErr bool
	}{
		{"http://localhost:8080", false},
		{"socks5://localhost:1080", false},
		{"invalid", true},
	}
	for _, tt := range tests {
		_, err := ParseUpstreamProxy(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseUpstreamProxy(%s) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
		}
	}
}
