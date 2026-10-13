package storage

import (
	"fmt"
	"sync"
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

func TestARCCacheTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		actions  func(c *ARCCache)
		checks   func(c *ARCCache, t *testing.T)
	}{
		{
			name:     "Basic Set and Get",
			capacity: 3,
			actions: func(c *ARCCache) {
				c.Set("k1", "v1")
				c.Set("k2", "v2")
			},
			checks: func(c *ARCCache, t *testing.T) {
				if v, ok := c.Get("k1"); !ok || v != "v1" {
					t.Errorf("expected v1, got %v", v)
				}
			},
		},
		{
			name:     "Capacity Boundary and Ghost List Adaptation",
			capacity: 2,
			actions: func(c *ARCCache) {
				c.Set("1", "val1")
				c.Set("2", "val2")
				// Access "1" to promote to T2
				c.Get("1")
				// Insert "3" triggering eviction of "2" into B1
				c.Set("3", "val3")
				// Re-insert "2" which hits B1 ghost list
				c.Set("2", "val2-new")
			},
			checks: func(c *ARCCache, t *testing.T) {
				if v, ok := c.Get("2"); !ok || v != "val2-new" {
					t.Errorf("expected val2-new, got %v", v)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewARCCache(tt.capacity)
			tt.actions(c)
			tt.checks(c, t)
		})
	}
}

func TestARCCacheConcurrency(t *testing.T) {
	t.Parallel()
	cache := NewARCCache(100)
	var wg sync.WaitGroup

	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("w-%d-k-%d", w, j)
				cache.Set(key, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()
}
