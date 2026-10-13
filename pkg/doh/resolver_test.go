package doh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestResolverTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		status      int
		expectError bool
	}{
		{"Valid", "example.com", http.StatusOK, false},
		{"InvalidDomain", "bad;domain", 0, true},
		{"UpstreamError", "example.com", http.StatusInternalServerError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte("dns-data"))
			}))
			defer ts.Close()
			r := NewResolver([]string{ts.URL})
			_, err := r.Resolve(context.Background(), tt.domain)
			if (err != nil) != tt.expectError {
				t.Errorf("Resolve() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestBalancer(t *testing.T) {
	b := NewBalancer([]string{"a", "b"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); b.Next() }()
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
