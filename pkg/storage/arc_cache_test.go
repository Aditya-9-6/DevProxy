package storage

import (
	"testing"
)

func TestARCCache(t *testing.T) {
	cache := NewARCCache(2)

	cache.Set("a", 1)
	cache.Set("b", 2)

	if val, ok := cache.Get("a"); !ok || val != 1 {
		t.Errorf("expected 1, got %v", val)
	}

	cache.Set("c", 3)
	if _, ok := cache.Get("b"); ok {
		t.Error("expected b to be evicted")
	}
}
