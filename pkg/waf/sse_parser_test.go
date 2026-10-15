package waf

import (
	"bytes"
	"sync"
	"testing"
)

func TestSSEParser_ParseChunk(t *testing.T) {
	parser := NewSSEParser()
	var events []SSEEvent
	parser.ParseChunk([]byte("event: message\nid: 1\ndata: hello\n\n"), func(e SSEEvent) { events = append(events, e) })
	if len(events) != 1 || !bytes.Equal(events[0].Data, []byte("hello\n")) {
		t.Errorf("failed parsing")
	}
}

func TestSSEParser_Streaming(t *testing.T) {
	parser := NewSSEParser()
	var events []SSEEvent
	parser.ParseChunk([]byte("data: part1\n"), func(e SSEEvent) { events = append(events, e) })
	parser.ParseChunk([]byte("data: part2\n\n"), func(e SSEEvent) { events = append(events, e) })
	if len(events) != 1 || !bytes.Equal(events[0].Data, []byte("part1\npart2\n")) {
		t.Errorf("failed streaming: %s", events[0].Data)
	}
}

func TestSSEParser_MemoryStability(t *testing.T) {
	parser := NewSSEParser()
	var captured []byte
	parser.ParseChunk([]byte("data: stable\n\n"), func(e SSEEvent) {
		captured = e.Data
	})
	parser.ParseChunk([]byte("data: overwrite\n\n"), func(e SSEEvent) {})
	if !bytes.Equal(captured, []byte("stable\n")) {
		t.Errorf("memory aliasing detected: got %s", captured)
	}
}

func TestSSEParser_Concurrency(t *testing.T) {
	parser := NewSSEParser()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser.ParseChunk([]byte("data: test\n\n"), func(e SSEEvent) {})
		}()
	}
	wg.Wait()
}

func TestSSEParser_MaxBufferExceeded(t *testing.T) {
	parser := NewSSEParserWithLimit(5)
	if err := parser.ParseChunk([]byte("data: too long\n\n"), func(e SSEEvent) {}); err != ErrMaxBufferExceeded {
		t.Errorf("expected error")
	}
}

func TestSSEParser_Closed(t *testing.T) {
	parser := NewSSEParser()
	parser.Close()
	if err := parser.ParseChunk([]byte("data: test\n\n"), func(e SSEEvent) {}); err != ErrParserClosed {
		t.Errorf("expected ErrParserClosed")
	}
}
