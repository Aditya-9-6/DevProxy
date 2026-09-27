package ringbuffer

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_PushAndPop(t *testing.T) {
	rb := NewRingBuffer(64)
	defer rb.Close()

	event := &TrafficEvent{
		ID:       "req-1",
		Host:     "localhost",
		Method:   "GET",
		URL:      "http://localhost:8080/test",
		Duration: 5 * time.Millisecond,
	}

	if !rb.Push(event) {
		t.Fatalf("Push failed")
	}

	popped := rb.Pop()
	if popped == nil || popped.ID != "req-1" {
		t.Fatalf("Expected event req-1, got %v", popped)
	}
}

func TestRingBuffer_SaturationNonBlocking(t *testing.T) {
	// Ring buffer with small size 16
	rb := NewRingBuffer(16)
	defer rb.Close()

	// Fill buffer completely
	for i := 0; i < 16; i++ {
		pushed := rb.Push(&TrafficEvent{ID: fmt.Sprintf("req-%d", i)})
		if !pushed {
			t.Fatalf("Expected push %d to succeed", i)
		}
	}

	// Next push should NOT block, but return false and increment dropped counter
	pushed := rb.Push(&TrafficEvent{ID: "overflow-1"})
	if pushed {
		t.Fatalf("Expected overflow push to be dropped")
	}

	queued, dropped, total := rb.Stats()
	if dropped != 1 {
		t.Fatalf("Expected 1 dropped event, got %d", dropped)
	}
	if queued != 16 {
		t.Fatalf("Expected 16 queued, got %d", queued)
	}
	if total != 16 {
		t.Fatalf("Expected total pushed 16, got %d", total)
	}
}

func TestRingBuffer_ConcurrentLoad(t *testing.T) {
	rb := NewRingBuffer(1024)
	defer rb.Close()

	var wg sync.WaitGroup
	producers := 8
	itemsPerProducer := 500

	// Multiple concurrent producers
	for p := 0; p < producers; p++ {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				rb.Push(&TrafficEvent{
					ID:     fmt.Sprintf("p%d-item%d", pid, i),
					Method: "GET",
				})
			}
		}(p)
	}

	// Concurrent consumer
	consumedCount := 0
	done := make(chan bool)
	go func() {
		for {
			item := rb.TryPop()
			if item != nil {
				consumedCount++
			} else {
				time.Sleep(100 * time.Microsecond)
			}
			select {
			case <-done:
				// Drain remainder
				for rb.TryPop() != nil {
					consumedCount++
				}
				return
			default:
			}
		}
	}()

	wg.Wait()
	time.Sleep(10 * time.Millisecond)
	close(done)

	_, dropped, total := rb.Stats()
	if total == 0 {
		t.Fatalf("Expected events to be pushed, got 0")
	}
	t.Logf("Concurrent Test: Pushed=%d, Dropped=%d, Consumed=%d", total, dropped, consumedCount)
}
