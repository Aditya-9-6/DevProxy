package header

import (
	"net/http"
	"sync"
	"testing"
)

func TestHeaderDeduplication(t *testing.T) {
	sm := NewStateMachine()
	h := http.Header{"Content-Type": []string{"application/json"}}
	hash := ComputeHeaderHash(h)

	b1 := sm.GetOrStore(hash, []byte("application/json"))
	b2 := sm.GetOrStore(hash, []byte("application/json"))

	if b1 != b2 {
		t.Errorf("Expected same pointer for identical header hash, got %p and %p", b1, b2)
	}
}

func TestHeaderDeduplication_Concurrency(t *testing.T) {
	t.Parallel()
	sm := NewStateMachine()
	hash := uint64(12345)
	data := []byte("test")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.GetOrStore(hash, data)
		}()
	}
	wg.Wait()
}

func BenchmarkGetOrStore(b *testing.B) {
	sm := NewStateMachine()
	hash := uint64(12345)
	data := []byte("test")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sm.GetOrStore(hash, data)
		}
	})
}
