package proxy

import (
	"sync"
	"testing"
)

func TestHeaderInterner(t *testing.T) {
	interner := NewHeaderInterner()

	k1 := "Content-Type"
	k2 := "Content-Type"

	val1 := interner.Intern(k1)
	val2 := interner.Intern(k2)

	if val1 != val2 {
		t.Errorf("Expected interned strings to be equal")
	}

	if val1 != "Content-Type" {
		t.Errorf("Expected Content-Type, got %s", val1)
	}
}

func TestHeaderInternerConcurrent(t *testing.T) {
	interner := NewHeaderInterner()
	const workers = 10
	const iterations = 1000
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				interner.Intern("Header-Key")
			}
		}()
	}
	wg.Wait()
}

func BenchmarkHeaderInterner(b *testing.B) {
	interner := NewHeaderInterner()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			interner.Intern("X-Request-ID")
		}
	})
}
