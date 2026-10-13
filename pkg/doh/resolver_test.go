package doh

import (
	"context"
	"testing"
)

func TestBalancer(t *testing.T) {
	b := NewBalancer([]string{"a", "b"})
	first := b.Next().URL
	second := b.Next().URL
	if first == second {
		t.Errorf("expected round robin, got same: %s", first)
	}
}

func TestResolverValidation(t *testing.T) {
	r := NewResolver([]string{"https://dns.google/dns-query"})
	_, err := r.Resolve(context.Background(), "invalid;domain")
	if err == nil {
		t.Error("expected error for invalid domain")
	}
}
