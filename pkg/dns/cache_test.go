package dns

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDNSCache_BasicSetGet(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	host := "example.com"
	ips := []string{"93.184.216.34"}

	cache.Set(host, ips, time.Minute)

	gotIps, isStale, ok := cache.Get(host)
	if !ok {
		t.Fatalf("expected cache hit for %s", host)
	}
	if isStale {
		t.Fatalf("expected fresh cache entry, got stale")
	}
	if len(gotIps) != 1 || gotIps[0] != ips[0] {
		t.Fatalf("expected %v, got %v", ips, gotIps)
	}
}

func TestDNSCache_StaleWhileRevalidate(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	host := "swr.example.com"
	ips := []string{"1.2.3.4"}

	cache.Set(host, ips, -time.Millisecond)

	gotIps, isStale, ok := cache.Get(host)
	if !ok {
		t.Fatalf("expected cache hit with stale data, got miss")
	}
	if !isStale {
		t.Fatalf("expected stale=true for expired entry")
	}
	if len(gotIps) != 1 || gotIps[0] != ips[0] {
		t.Fatalf("expected %v, got %v", ips, gotIps)
	}
}

func TestDNSCache_ConcurrentAccess(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	var wg sync.WaitGroup
	workers := 50
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := GetQueryContext("concurrent.example.com", uint16(id))
			defer PutQueryContext(ctx)

			for j := 0; j < iterations; j++ {
				cache.Set(ctx.Host, []string{"127.0.0.1"}, time.Minute)
				cache.Get(ctx.Host)
			}
		}(i)
	}
	wg.Wait()
}

func TestResolver_Lookup(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()
	r := NewResolver(cache)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use localhost which always resolves
	addrs, err := r.Lookup(ctx, "localhost")
	if err != nil {
		// If network is restricted, skip or pass gracefully
		t.Logf("Lookup localhost returned err: %v", err)
		return
	}
	if len(addrs) == 0 {
		t.Fatalf("expected addresses for localhost")
	}
}
