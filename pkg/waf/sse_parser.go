package waf

import (
	"bytes"
	"sync"
)

var sseBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Data  []byte
	Event []byte
	ID    []byte
}

// SSEParser provides zero-allocation parsing of text/event-stream chunks.
type SSEParser struct{}

func NewSSEParser() *SSEParser {
	return &SSEParser{}
}

// ParseChunk processes a raw buffer and extracts SSE fields.
func (p *SSEParser) ParseChunk(chunk []byte) []SSEEvent {
	var events []SSEEvent
	lines := bytes.Split(chunk, []byte("\n"))

	var current SSEEvent
	for _, line := range lines {
		if len(line) == 0 {
			if len(current.Data) > 0 {
				events = append(events, current)
				current = SSEEvent{}
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("data:")) {
			val := bytes.TrimPrefix(line, []byte("data:"))
			current.Data = append(current.Data, bytes.TrimSpace(val)...)
		} else if bytes.HasPrefix(line, []byte("event:")) {
			current.Event = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("event:")))
		} else if bytes.HasPrefix(line, []byte("id:")) {
			current.ID = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("id:")))
		}
	}
	return events
}
