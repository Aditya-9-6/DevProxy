package dns

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDNSCacheSetGet(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	host := "example.com"
	ips := []string{"192.0.2.1", "192.0.2.2"}

	// Test getting non-existent
	if _, ok := cache.Get(host); ok {
		t.Error("Expected false for non-existent host")
	}

	// Test setting and getting
	cache.Set(host, ips, 1*time.Minute)

	gotIPs, ok := cache.Get(host)
	if !ok {
		t.Error("Expected true for existing host")
	}

	if !reflect.DeepEqual(gotIPs, ips) {
		t.Errorf("Expected IPs %v, got %v", ips, gotIPs)
	}
}

func TestDNSCacheExpiry(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	host := "example.com"
	ips := []string{"192.0.2.1"}

	// Set with very short TTL
	cache.Set(host, ips, 10*time.Millisecond)

	// Verify it's there
	if _, ok := cache.Get(host); !ok {
		t.Fatal("Expected entry to be present initially")
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Verify it's gone
	if _, ok := cache.Get(host); ok {
		t.Error("Expected entry to be expired")
	}
}

func TestDNSCacheConcurrency(t *testing.T) {
	cache := NewDNSCache()
	defer cache.Close()

	var wg sync.WaitGroup
	workers := 100
	iterations := 100

	// Concurrent writes
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Set("example.com", []string{"192.0.2.1"}, 1*time.Minute)
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Get("example.com")
			}
		}()
	}

	wg.Wait()
}

func TestDNSCacheClose(t *testing.T) {
	cache := NewDNSCache()

	// Close should not panic and should cancel the context
	cache.Close()

	// Calling close again should be safe
	cache.Close()
}
