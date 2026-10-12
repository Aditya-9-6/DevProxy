package stream

import (
	"sync"
	"testing"
)

func TestJitterBuffer_Resequencing(t *testing.T) {
	jb := NewJitterBuffer()

	// Push out of order
	if !jb.Push(1, []byte("data1")) {
		t.Error("Failed to push seq 1")
	}
	if !jb.Push(0, []byte("data0")) {
		t.Error("Failed to push seq 0")
	}

	// Pop in order
	p0 := jb.Pop()
	if p0 == nil || string(p0.Data) != "data0" {
		t.Errorf("Expected data0, got %v", p0)
	}
	jb.Recycle(p0)

	p1 := jb.Pop()
	if p1 == nil || string(p1.Data) != "data1" {
		t.Errorf("Expected data1, got %v", p1)
	}
	jb.Recycle(p1)
}

func TestJitterBuffer_Empty(t *testing.T) {
	jb := NewJitterBuffer()
	if p := jb.Pop(); p != nil {
		t.Error("Expected nil on empty buffer")
	}
}

func TestJitterBuffer_WindowValidation(t *testing.T) {
	jb := NewJitterBuffer()

	// Push within window (nextSeq = 0)
	if !jb.Push(0, []byte("zero")) {
		t.Error("Expected seq 0 to be accepted")
	}

	// Push too far ahead (nextSeq + BufferSize)
	if jb.Push(BufferSize+1, []byte("too-far")) {
		t.Error("Expected seq beyond window to be rejected")
	}

	// Consume seq 0
	p := jb.Pop()
	jb.Recycle(p)

	// Now seq BufferSize should be within window
	if !jb.Push(BufferSize, []byte("edge")) {
		t.Error("Expected seq BufferSize to be accepted after advance")
	}
}

func TestJitterBuffer_WrapAround(t *testing.T) {
	jb := NewJitterBuffer()

	var seq uint64
	for seq = 0; seq < BufferSize*2; seq++ {
		data := []byte("packet")
		if !jb.Push(seq, data) {
			t.Fatalf("Failed to push at seq %d", seq)
		}
		p := jb.Pop()
		if p == nil || p.Sequence != seq {
			t.Fatalf("Failed to pop correct sequence at %d, got %v", seq, p)
		}
		jb.Recycle(p)
	}
}

func TestJitterBuffer_Concurrent(t *testing.T) {
	jb := NewJitterBuffer()
	var wg sync.WaitGroup
	workers := 4
	packetsPerWorker := 250

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < packetsPerWorker; i++ {
				seq := uint64(workerID*packetsPerWorker + i)
				for !jb.Push(seq, []byte("payload")) {
					// Retry until accepted by sliding window
				}
			}
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		consumed := 0
		maxPackets := workers * packetsPerWorker
		for consumed < maxPackets {
			if p := jb.Pop(); p != nil {
				consumed++
				jb.Recycle(p)
			}
		}
	}()

	wg.Wait()
}
