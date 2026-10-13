package doh

import (
	"sync/atomic"
)

// Upstream represents a DoH server endpoint.
type Upstream struct {
	URL    string
	Active atomic.Uint64
}

// Balancer manages DoH upstream servers using atomic round-robin.
type Balancer struct {
	upstreams []*Upstream
	index     atomic.Uint64
}

func NewBalancer(urls []string) *Balancer {
	upstreams := make([]*Upstream, len(urls))
	for i, url := range urls {
		upstreams[i] = &Upstream{URL: url}
	}
	return &Balancer{upstreams: upstreams}
}

func (b *Balancer) Next() *Upstream {
	if len(b.upstreams) == 0 {
		return nil
	}
	idx := b.index.Add(1) - 1
	return b.upstreams[idx%uint64(len(b.upstreams))]
}
