package waf

import (
	"bytes"
	"testing"
)

func TestSSEParser_ParseChunk(t *testing.T) {
	parser := NewSSEParser()
	chunk := []byte("event: message\nid: 1\ndata: hello world\n\ndata: next event\n\n")

	events := parser.ParseChunk(chunk)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if !bytes.Equal(events[0].Data, []byte("hello world")) {
		t.Errorf("expected data 'hello world', got %s", events[0].Data)
	}

	if !bytes.Equal(events[1].Data, []byte("next event")) {
		t.Errorf("expected data 'next event', got %s", events[1].Data)
	}
}
