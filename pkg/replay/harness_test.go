package replay

import (
	"context"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"testing"
	"time"
)

func TestHarnessReplay(t *testing.T) {
	h := NewHarness(8)
	event := &ringbuffer.TrafficEvent{ID: "test", Timestamp: time.Now()}
	h.buffer.Push(event)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	found := false
	h.Replay(ctx, func(e *ringbuffer.TrafficEvent) {
		found = true
	})

	if !found {
		t.Error("Expected event not replayed")
	}
}
