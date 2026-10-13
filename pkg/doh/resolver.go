package doh

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Resolver performs DoH lookups.
type Resolver struct {
	balancer *Balancer
	cache    *Cache
	client   *http.Client
}

func NewResolver(urls []string) *Resolver {
	return &Resolver{
		balancer: NewBalancer(urls),
		cache:    NewCache(),
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (r *Resolver) Resolve(ctx context.Context, domain string) ([]byte, error) {
	if cached, ok := r.cache.Get(domain); ok {
		return cached, nil
	}

	upstream := r.balancer.Next()
	req, _ := http.NewRequestWithContext(ctx, "GET", upstream.URL+"?name="+domain, nil)
	req.Header.Set("Accept", "application/dns-message")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	r.cache.Set(domain, body, 5*time.Minute)
	return body, nil
}
