package proxy

import (
	"context"
	"sync"
	"testing"
)

func TestPipelineQueue_NormalOperation(t *testing.T) {
	pq := NewPipelineQueue(4)
	defer pq.Close()

	req := pq.AcquireRequest()
	req.ID = 1
	req.Payload = []byte("GET / HTTP/1.1")
	req.Context = context.Background()

	if err := pq.Enqueue(req); err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	if pq.Inflight() != 1 {
		t.Fatalf("expected inflight 1, got %d", pq.Inflight())
	}

	dequeued, err := pq.Dequeue()
	if err != nil {
		t.Fatalf("failed to dequeue: %v", err)
	}

	if dequeued.ID != 1 || string(dequeued.Payload) != "GET / HTTP/1.1" {
		t.Fatalf("unexpected payload: %+v", dequeued)
	}

	pq.ReleaseRequest(dequeued)

	if pq.Inflight() != 0 {
		t.Fatalf("expected inflight 0 after dequeue, got %d", pq.Inflight())
	}
}

func TestPipelineQueue_BackpressureAndOverflow(t *testing.T) {
	pq := NewPipelineQueue(2)
	defer pq.Close()

	r1 := pq.AcquireRequest()
	r1.ID = 1
	r2 := pq.AcquireRequest()
	r2.ID = 2
	r3 := pq.AcquireRequest()
	r3.ID = 3

	if err := pq.Enqueue(r1); err != nil {
		t.Fatalf("unexpected error on r1: %v", err)
	}
	if err := pq.Enqueue(r2); err != nil {
		t.Fatalf("unexpected error on r2: %v", err)
	}

	// Should trigger backpressure limit
	err := pq.Enqueue(r3)
	if err != ErrPipelineFull {
		t.Fatalf("expected ErrPipelineFull, got %v", err)
	}

	// Dequeue one to free space
	d, err := pq.Dequeue()
	if err != nil {
		t.Fatalf("unexpected dequeue error: %v", err)
	}
	pq.ReleaseRequest(d)

	// Now r3 should succeed
	if err := pq.Enqueue(r3); err != nil {
		t.Fatalf("expected success after freeing space, got %v", err)
	}
}

func TestPipelineQueue_ConcurrentPipelining(t *testing.T) {
	pq := NewPipelineQueue(1024)
	defer pq.Close()

	var wg sync.WaitGroup
	workers := 4
	itemsPerWorker := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWorker; j++ {
				req := pq.AcquireRequest()
				req.ID = uint64(workerID*1000 + j)
				req.Payload = []byte("HTTP/1.1 POST")
				for {
					if err := pq.Enqueue(req); err == nil {
						break
					}
				}
			}
		}(i)
	}

	wg.Wait()

	consumed := 0
	for {
		dequeued, err := pq.Dequeue()
		if err != nil {
			break
		}
		pq.ReleaseRequest(dequeued)
		consumed++
	}

	expectedCount := workers * itemsPerWorker
	if consumed != expectedCount {
		t.Fatalf("expected %d consumed items, got %d", expectedCount, consumed)
	}
}
