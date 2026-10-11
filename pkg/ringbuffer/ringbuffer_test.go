package ringbuffer

import (
	"sync"
	"testing"
)

func TestRingBuffer_Concurrency(t *testing.T) {
	buf := NewRingBuffer(1024)
	const count = 10000
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for !buf.Push(&TrafficEvent{ID: "test"}) {
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for buf.Pop() == nil {
			}
		}
	}()

	wg.Wait()
}
