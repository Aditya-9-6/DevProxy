package doh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

// Resolver performs DoH lookups.
type Resolver struct {
	balancer *Balancer
	cache    *Cache
	client   *http.Client
}

func NewResolver(urls []string) *Resolver {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &Resolver{
		balancer: NewBalancer(urls),
		cache:    NewCache(),
		client:   &http.Client{Transport: transport, Timeout: 5 * time.Second},
	}
}

func (r *Resolver) Resolve(ctx context.Context, domain string) ([]byte, error) {
	if !domainRegex.MatchString(domain) {
		return nil, errors.New("invalid domain format")
	}

	if cached, ok := r.cache.Get(domain); ok {
		return cached, nil
	}

	upstream := r.balancer.Next()
	rawURL := fmt.Sprintf("%s?name=%s", upstream.URL, url.QueryEscape(domain))
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-message")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := r.cache.pool.Get().([]byte)
	n, err := io.ReadAtLeast(resp.Body, buf, 0)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		r.cache.pool.Put(buf)
		return nil, err
	}

	data := make([]byte, n)
	copy(data, buf[:n])
	r.cache.pool.Put(buf)

	r.cache.Set(domain, data, 5*time.Minute)
	return data, nil
}
