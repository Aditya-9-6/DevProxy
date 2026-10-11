package security

import (
	"sync"
	"testing"
)

func TestStreamReassembler(t *testing.T) {
	r := NewStreamReassembler(0)
	var result []byte
	var mu sync.Mutex
	callback := func(b []byte) { mu.Lock(); result = append(result, b...); mu.Unlock() }

	r.Push(0, []byte("hello"), callback)
	r.Push(5, []byte("world"), callback)

	if string(result) != "helloworld" {
		t.Errorf("expected helloworld, got %s", string(result))
	}
}

func TestConcurrentPush(t *testing.T) {
	t.Parallel()
	r := NewStreamReassembler(0)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.Push(uint64(i*5), []byte("data "), func(b []byte) {})
		}(i)
	}
	wg.Wait()
}

func TestMaxSegs(t *testing.T) {
	r := NewStreamReassembler(0)
	r.maxSegs = 1
	callback := func(b []byte) {}

	r.Push(10, []byte("data"), callback)
	r.Push(20, []byte("data"), callback)

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.segments.Len() > 1 {
		t.Error("maxSegs limit not respected")
	}
}
