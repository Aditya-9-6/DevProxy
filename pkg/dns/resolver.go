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
	qCtx := GetQueryContext(host, 1)
	defer PutQueryContext(qCtx)

	if ips, isStale, ok := r.cache.Get(qCtx.Host); ok {
		if isStale {
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if addrs, err := net.DefaultResolver.LookupHost(bgCtx, host); err == nil {
					r.cache.Set(host, addrs, 5*time.Minute)
				}
			}()
		}
		return ips, nil
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}
	r.cache.Set(host, addrs, 5*time.Minute)
	return addrs, nil
}
