package ringbuffer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchFlusher_ConcurrentFlush(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(1024)
	defer rb.Close()
	flusher := NewBatchFlusher(rb)

	var producers sync.WaitGroup
	for producer := 0; producer < 4; producer++ {
		producers.Add(1)
		go func(producerID int) {
			defer producers.Done()
			produceRingEvents(rb, producerID, 250)
		}(producer)
	}

	var totalFlushed int64
	var consumers sync.WaitGroup
	for consumer := 0; consumer < 4; consumer++ {
		consumers.Add(1)
		go func() {
			defer consumers.Done()
			flushRingBatches(flusher, &totalFlushed)
		}()
	}

	producers.Wait()
	consumers.Wait()

	if batch := flusher.Flush(); batch != nil {
		atomic.AddInt64(&totalFlushed, int64(len(batch)))
		flusher.Recycle(batch)
	}
	if atomic.LoadInt64(&totalFlushed) == 0 {
		t.Error("Expected events to be flushed via BatchFlusher, got 0")
	}
}

func TestRingBuffer_PushAndPop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "ordinary identifier", id: "req-1"},
		{name: "empty identifier", id: ""},
		{name: "unicode identifier", id: "请求-✓"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			rb := NewRingBuffer(4)
			defer rb.Close()

			event := &TrafficEvent{
				ID:       test.id,
				Host:     "localhost",
				Method:   "GET",
				URL:      "http://localhost:8080/test",
				Duration: 5 * time.Millisecond,
			}
			if !rb.Push(event) {
				t.Fatal("Push failed")
			}

			popped := rb.Pop()
			if popped == nil || popped.ID != test.id {
				t.Fatalf("Expected event %q, got %v", test.id, popped)
			}
		})
	}
}

func TestRingBuffer_SaturationNonBlocking(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(16)
	defer rb.Close()

	for index := 0; index < 16; index++ {
		if !rb.Push(&TrafficEvent{ID: fmt.Sprintf("req-%d", index)}) {
			t.Fatalf("Expected push %d to succeed", index)
		}
	}

	if rb.Push(&TrafficEvent{ID: "overflow-1"}) {
		t.Fatal("Expected overflow push to be dropped")
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
	t.Parallel()

	rb := NewRingBuffer(1024)
	defer rb.Close()

	var producers sync.WaitGroup
	for producer := 0; producer < 8; producer++ {
		producers.Add(1)
		go func(producerID int) {
			defer producers.Done()
			produceRingEvents(rb, producerID, 500)
		}(producer)
	}

	var consumed int64
	done := make(chan struct{})
	var consumer sync.WaitGroup
	consumer.Add(1)
	go func() {
		defer consumer.Done()
		consumeRingEvents(rb, done, &consumed)
	}()

	producers.Wait()
	time.Sleep(10 * time.Millisecond)
	close(done)
	consumer.Wait()

	_, dropped, total := rb.Stats()
	if total == 0 {
		t.Fatal("Expected events to be pushed, got 0")
	}
	t.Logf("Concurrent Test: Pushed=%d, Dropped=%d, Consumed=%d", total, dropped, atomic.LoadInt64(&consumed))
}

func produceRingEvents(rb *RingBuffer, producerID, count int) {
	for index := 0; index < count; index++ {
		rb.Push(&TrafficEvent{
			ID:     fmt.Sprintf("p%d-item%d", producerID, index),
			Method: "GET",
		})
	}
}

func flushRingBatches(flusher *BatchFlusher, total *int64) {
	for iteration := 0; iteration < 50; iteration++ {
		time.Sleep(2 * time.Millisecond)
		batch := flusher.Flush()
		if batch == nil {
			continue
		}
		atomic.AddInt64(total, int64(len(batch)))
		flusher.Recycle(batch)
	}
}

func consumeRingEvents(rb *RingBuffer, done <-chan struct{}, consumed *int64) {
	for {
		if event := rb.TryPop(); event != nil {
			atomic.AddInt64(consumed, 1)
			continue
		}

		select {
		case <-done:
			for rb.TryPop() != nil {
				atomic.AddInt64(consumed, 1)
			}
			return
		default:
			time.Sleep(10 * time.Microsecond)
		}
	}
}

func BenchmarkRingBuffer_PushTryPop(b *testing.B) {
	rb := NewRingBuffer(16384)
	defer rb.Close()
	event := &TrafficEvent{
		ID:     "bench-1",
		Host:   "example.com",
		Method: "GET",
		URL:    "http://example.com/api",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if rb.Push(event) {
			_ = rb.TryPop()
		}
	}
}

func BenchmarkRingBuffer_ParallelPush(b *testing.B) {
	rb := NewRingBuffer(65536)
	defer rb.Close()
	event := &TrafficEvent{ID: "bench-p", Host: "example.com"}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(parallel *testing.PB) {
		for parallel.Next() {
			rb.Push(event)
			_ = rb.TryPop()
		}
	})
}

func FuzzRingBuffer(f *testing.F) {
	f.Add([]byte("event-payload-1"), uint16(16))
	f.Add([]byte("large-event-payload-long-string-data"), uint16(128))
	f.Add([]byte("kanal"), uint16(1))

	f.Fuzz(func(t *testing.T, payload []byte, size uint16) {
		if len(payload) == 0 {
			return
		}
		capacity := int(size%256) + 1
		rb := NewRingBuffer(capacity)
		defer rb.Close()

		event := &TrafficEvent{
			ID:   string(payload),
			Host: "fuzz.host",
		}
		for index := 0; index < 32; index++ {
			rb.Push(&TrafficEvent{ID: fmt.Sprintf("%s-%d", payload, index), Host: "fuzz.host"})
		}
		for index := 0; index < 32; index++ {
			_ = rb.TryPop()
		}

		if rb.Push(event) {
			popped := rb.TryPop()
			if popped != nil && popped.ID != string(payload) {
				t.Fatalf("mismatched payload in ringbuffer: expected %s, got %s", payload, popped.ID)
			}
		}
	})
}
