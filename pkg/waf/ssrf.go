package waf

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync"
)

// SSRFEnforcer handles high-level SSRF protection logic.
type SSRFEnforcer struct {
	matcher *SSRFMatcher
	pool    sync.Pool
}

// NewSSRFEnforcer initializes an enforcer with default private network boundaries.
func NewSSRFEnforcer() *SSRFEnforcer {
	e := &SSRFEnforcer{
		matcher: NewSSRFMatcher(),
		pool: sync.Pool{
			New: func() interface{} {
				return make([]netip.Addr, 0, 16)
			},
		},
	}

	// Default RFC1918, Loopback, Link-Local, and IPv6 equivalents
	defaultCIDRs := []string{
		"127.0.0.0/8",    // IPv4 Loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // IPv4 Link-Local
		"::1/128",        // IPv6 Loopback
		"fe80::/10",      // IPv6 Link-Local
		"fc00::/7",       // IPv6 Unique Local Address (ULA)
	}

	prefixes := make([]netip.Prefix, 0, len(defaultCIDRs))
	for _, c := range defaultCIDRs {
		if p, err := netip.ParsePrefix(c); err == nil {
			prefixes = append(prefixes, p)
		}
	}
	e.matcher.UpdateBlocklist(prefixes)

	return e
}

// IsSafe checks if a target URL is safe from SSRF by resolving its hostname and checking IPs.
func (e *SSRFEnforcer) IsSafe(ctx context.Context, targetURL string) (bool, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false, fmt.Errorf("invalid url: %w", err)
	}

	host := u.Hostname()
	if host == "" {
		return false, nil
	}

	// 1. Check if host is a direct IP
	if addr, err := netip.ParseAddr(host); err == nil {
		if e.matcher.IsForbidden(addr) {
			return false, nil
		}
		return true, nil
	}

	// 2. Resolve hostname to IPs
	// Note: In a production proxy, we should use the resolved IPs for the actual connection
	// to prevent DNS rebinding attacks.
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		// If we can't resolve it, we can't verify it. For security, we might block,
		// but usually, we allow the dialer to fail later.
		return true, nil
	}

	for _, addr := range addrs {
		// Normalize IPv4-mapped IPv6 to IPv4
		if addr.Is4In6() {
			addr = addr.Unmap()
		}
		if e.matcher.IsForbidden(addr) {
			return false, nil
		}
	}

	return true, nil
}

// UpdateCustomBlocklist allows dynamic updates to the forbidden ranges.
func (e *SSRFEnforcer) UpdateCustomBlocklist(cidrs []string) error {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(strings.TrimSpace(c))
		if err != nil {
			return err
		}
		prefixes = append(prefixes, p)
	}
	e.matcher.UpdateBlocklist(prefixes)
	return nil
}
