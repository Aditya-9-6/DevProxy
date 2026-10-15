package resolver

import (
	"context"
	"net"
	"time"
)

type Resolver struct {
	cache *DNSCache
}

func NewResolver() *Resolver {
	return &Resolver{cache: NewDNSCache()}
}

func (r *Resolver) Resolve(ctx context.Context, host string) ([]string, error) {
	if ips, ok := r.cache.Get(host); ok {
		return ips, nil
	}
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		r.cache.Set(host, nil, 30*time.Second) // Negative caching
		return nil, err
	}
	r.cache.Set(host, addrs, 5*time.Minute)
	return addrs, nil
}
