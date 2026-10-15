package waf

import (
	"bytes"
	"testing"
)

func TestSSEParser_ParseChunk(t *testing.T) {
	parser := NewSSEParser()
	chunk := []byte("event: message\nid: 1\ndata: hello world\n\ndata: next event\n\n")

	events, err := parser.ParseChunk(chunk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func TestSSEParser_FragmentedChunks(t *testing.T) {
	parser := NewSSEParser()

	// Send first part of event
	events1, err := parser.ParseChunk([]byte("event: update\ndata: part1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events1) != 0 {
		t.Fatalf("expected 0 events for partial chunk, got %d", len(events1))
	}

	// Send remaining part of event plus terminator
	events2, err := parser.ParseChunk([]byte("\ndata: part2\n\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events2) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events2))
	}

	expectedData := []byte("part1\npart2")
	if !bytes.Equal(events2[0].Data, expectedData) {
		t.Errorf("expected data %q, got %q", expectedData, events2[0].Data)
	}
	if !bytes.Equal(events2[0].Event, []byte("update")) {
		t.Errorf("expected event 'update', got %s", events2[0].Event)
	}
}

func TestSSEParser_MaxBufferExceeded(t *testing.T) {
	parser := NewSSEParserWithLimit(20)
	chunk := []byte("data: this is a very long payload that exceeds limit\n\n")

	_, err := parser.ParseChunk(chunk)
	if err != ErrMaxBufferExceeded {
		t.Errorf("expected ErrMaxBufferExceeded, got %v", err)
	}
}
