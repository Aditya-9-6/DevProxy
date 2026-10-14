package tls

import (
	"testing"
	"time"
)

func TestOCSPCache(t *testing.T) {
	cache := NewOCSPCache()
	host := "example.com"
	staple := &OCSPStaple{Response: []byte("data"), Expiry: time.Now().Add(time.Hour)}

	cache.Set(host, staple)
	got := cache.Get(host)

	if got == nil || string(got.Response) != "data" {
		t.Errorf("expected data, got %v", got)
	}
}
