package proxy

import (
	"testing"
)

func TestNewFramePool(t *testing.T) {
	pool := NewFramePool()
	if pool == nil {
		t.Fatal("NewFramePool() returned nil")
	}
	if len(pool.pools) != 3 {
		t.Errorf("Expected 3 pools, got %d", len(pool.pools))
	}
	expectedSizes := []int{16384, 32768, 65536}
	for _, size := range expectedSizes {
		if _, ok := pool.pools[size]; !ok {
			t.Errorf("Missing pool for size %d", size)
		}
	}
}

func TestFramePoolGet(t *testing.T) {
	pool := NewFramePool()

	tests := []struct {
		name         string
		requestSize  int
		expectedSize int
	}{
		{"Small size", 1000, 16384},
		{"Exact small slab", 16384, 16384},
		{"Medium size", 20000, 32768},
		{"Exact medium slab", 32768, 32768},
		{"Large size", 50000, 65536},
		{"Exact large slab", 65536, 65536},
		{"Extra large size", 100000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := pool.Get(tt.requestSize)
			if cap(b) != tt.expectedSize {
				t.Errorf("Get(%d) returned capacity %d, expected %d", tt.requestSize, cap(b), tt.expectedSize)
			}
			if len(b) != cap(b) {
				t.Errorf("Get(%d) returned length %d, expected %d", tt.requestSize, len(b), cap(b))
			}
		})
	}
}

func TestFramePoolPut(t *testing.T) {
	pool := NewFramePool()

	// Test putting a valid buffer back
	b1 := pool.Get(1000)
	if cap(b1) != 16384 {
		t.Fatalf("Expected capacity 16384, got %d", cap(b1))
	}

	// Modify the buffer to ensure it is reset when retrieved
	b1[0] = 1

	pool.Put(b1)

	// Since it's a sync.Pool, there's no guarantee we get the same buffer back immediately
	// across garbage collections, but in a simple test without GC, we usually do.
	// But we're just checking that Put doesn't crash here and correctly handles the slices.

	b2 := pool.Get(1000)
	if cap(b2) != 16384 {
		t.Fatalf("Expected capacity 16384, got %d", cap(b2))
	}

	// Test putting an unsupported size back
	b3 := make([]byte, 100000)
	pool.Put(b3) // Should ignore without panicking
}
