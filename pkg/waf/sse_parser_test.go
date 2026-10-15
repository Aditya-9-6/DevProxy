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
