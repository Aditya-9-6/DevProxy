package dns

import (
	"context"
	"net"
	"time"
)

type Resolver struct {
	cache *DNSCache
}

func NewResolver(cache *DNSCache) *Resolver {
	return &Resolver{cache: cache}
}

func (r *Resolver) Lookup(ctx context.Context, host string) ([]string, error) {
	if ips, ok := r.cache.Get(host); ok {
		return ips, nil
	}

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 2 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}

	ips, err := resolver.LookupHost(ctx, host)
	if err == nil {
		r.cache.Set(host, ips, 5*time.Minute)
	}
	return ips, err
}
