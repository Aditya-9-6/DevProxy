package waf

import (
	"context"
	"net/netip"
	"testing"
)

func TestSSRFMatcher_IsForbidden(t *testing.T) {
	matcher := NewSSRFMatcher()
	prefixes := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.168.1.0/24"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fe80::/10"),
	}
	matcher.UpdateBlocklist(prefixes)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"127.255.255.255", true},
		{"10.0.0.1", true},
		{"10.255.255.254", true},
		{"192.168.1.50", true},
		{"192.168.2.1", false},
		{"8.8.8.8", false},
		{"::1", true},
		{"fe80::1", true},
		{"2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			addr := netip.MustParseAddr(tt.ip)
			got := matcher.IsForbidden(addr)
			if got != tt.expected {
				t.Errorf("IsForbidden(%s) = %v; want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestSSRFEnforcer_IsSafe(t *testing.T) {
	enforcer := NewSSRFEnforcer()
	ctx := context.Background()

	tests := []struct {
		url      string
		expected bool
	}{
		{"http://127.0.0.1/admin", false},
		{"https://10.0.0.1:8080/", false},
		{"http://localhost/", false}, // localhost resolves to 127.0.0.1
		{"https://google.com/", true},
		{"http://8.8.8.8/", true},
		{"http://[::1]/", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			safe, err := enforcer.IsSafe(ctx, tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if safe != tt.expected {
				t.Errorf("IsSafe(%s) = %v; want %v", tt.url, safe, tt.expected)
			}
		})
	}
}

func BenchmarkSSRFMatcher_IsForbidden(b *testing.B) {
	matcher := NewSSRFMatcher()
	matcher.UpdateBlocklist([]netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
	})
	addr := netip.MustParseAddr("192.168.1.1")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = matcher.IsForbidden(addr)
		// This should result in 0 B/op
	}
}
