package ringbuffer

import (
	"fmt"
	"testing"
)

func TestRingBuffer_SaturationNonBlocking(t *testing.T) {
	rb := NewRingBuffer(16)
	defer rb.Close()
	for i := 0; i < 16; i++ {
		rb.Push(&TrafficEvent{ID: fmt.Sprintf("req-%d", i)})
	}
	pushed := rb.Push(&TrafficEvent{ID: "overflow-1"})
	if pushed {
		t.Fatal("Expected overflow push to be dropped")
	}
	queued, dropped, total := rb.Stats()
	if dropped != 1 || queued != 16 || total != 17 {
		t.Fatalf("Expected 1 dropped, 16 queued, 17 total, got %d, %d, %d", dropped, queued, total)
	}
}
