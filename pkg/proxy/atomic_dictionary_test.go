package proxy

import (
	"fmt"
	"sync"
	"testing"
)

func TestAtomicDictionary(t *testing.T) {
	dict := NewAtomicDictionary()
	dict.Put("Content-Type", "application/json")
	dict.Put("X-Request-ID", "abc-123")

	val, ok := dict.Get("Content-Type")
	if !ok || val != "application/json" {
		t.Errorf("Expected application/json, got %s", val)
	}

	_, ok = dict.Get("Non-Existent")
	if ok {
		t.Error("Expected false for non-existent key")
	}
}

func TestAtomicDictionary_Concurrent(t *testing.T) {
	t.Parallel()
	dict := NewAtomicDictionary()

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("header-%d", j)
				val := fmt.Sprintf("val-%d-%d", workerID, j)
				dict.Put(key, val)
				_, _ = dict.Get(key)
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkAtomicDictionary_Get(b *testing.B) {
	dict := NewAtomicDictionary()
	dict.Put("k", "v")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dict.Get("k")
	}
}
