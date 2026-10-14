package resolver

import (
	"testing"
	"time"
)

func TestDNSCache(t *testing.T) {
	c := NewDNSCache()
	host := "example.com"
	ips := []string{"1.1.1.1", "8.8.8.8"}

	c.Set(host, ips, 1*time.Second)

	got, ok := c.Get(host)
	if !ok || len(got) != 2 {
		t.Errorf("expected %v, got %v", ips, got)
	}

	time.Sleep(1100 * time.Millisecond)
	_, ok = c.Get(host)
	if ok {
		t.Error("expected cache miss after TTL")
	}
}
