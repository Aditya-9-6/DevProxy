package proxy

import (
	"sync"
	"testing"
)

func TestSocketMap_Concurrency(t *testing.T) {
	sm := NewSocketMap()
	const iterations = 1000
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			sm.Set("key", "target")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			sm.Get("key")
		}
	}()

	wg.Wait()
}

func TestSocketMap_Logic(t *testing.T) {
	sm := NewSocketMap()
	tests := []struct {
		name   string
		key    string
		target string
		want   string
	}{{"happy", "a", "b", "b"}, {"notfound", "c", "", ""}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target != "" {
				sm.Set(tt.key, tt.target)
			}
			if got := sm.Get(tt.key); got != tt.want {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
