package doh

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBalancer(t *testing.T) {
	b := NewBalancer([]string{"a", "b"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Next()
		}()
	}
	wg.Wait()
}

func TestCacheConcurrency(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) { defer wg.Done(); c.Set("key", []byte("val"), time.Minute) }(i)
		go func() { defer wg.Done(); c.Get("key") }()
	}
	wg.Wait()
}

func TestResolverValidation(t *testing.T) {
	r := NewResolver([]string{"https://dns.google/dns-query"})
	_, err := r.Resolve(context.Background(), "invalid;domain")
	if err == nil {
		t.Error("expected error for invalid domain")
	}
}

func TestContextCancellation(t *testing.T) {
	r := NewResolver([]string{"https://dns.google/dns-query"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Resolve(ctx, "google.com")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
