package waf

import (
	"bytes"
	"errors"
	"sync"
)

var sseBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4096)
	},
}

// ErrMaxBufferExceeded is returned when an SSE stream exceeds the maximum allowed buffer size.
var ErrMaxBufferExceeded = errors.New("sse: maximum buffer size exceeded")

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Data  []byte
	Event []byte
	ID    []byte
}

// SSEParser provides zero-allocation parsing of text/event-stream chunks with state tracking.
type SSEParser struct {
	buffer    []byte
	current   SSEEvent
	maxBuffer int
}

// NewSSEParser creates a new SSEParser with default settings.
func NewSSEParser() *SSEParser {
	return &SSEParser{
		maxBuffer: 1024 * 1024, // 1MB limit by default
		buffer:    sseBufferPool.Get().([]byte),
	}
}

// NewSSEParserWithLimit creates a new SSEParser with a custom max buffer limit.
func NewSSEParserWithLimit(maxBuffer int) *SSEParser {
	return &SSEParser{
		maxBuffer: maxBuffer,
		buffer:    sseBufferPool.Get().([]byte),
	}
}

// Close releases the internal buffer back to the pool.
func (p *SSEParser) Close() {
	if p.buffer != nil {
		// Reset buffer before returning to pool
		poolBuf := p.buffer[:0]
		sseBufferPool.Put(poolBuf)
		p.buffer = nil
	}
}

// ParseChunk processes a raw buffer chunk and extracts SSE fields, handling partial chunks.
func (p *SSEParser) ParseChunk(chunk []byte) ([]SSEEvent, error) {
	if len(p.buffer)+len(chunk) > p.maxBuffer {
		return nil, ErrMaxBufferExceeded
	}

	p.buffer = append(p.buffer, chunk...)
	var events []SSEEvent

	start := 0
	bufLen := len(p.buffer)

	for start < bufLen {
		idx := bytes.IndexByte(p.buffer[start:], '\n')
		if idx == -1 {
			break
		}

		lineEnd := start + idx
		line := p.buffer[start:lineEnd]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		start = lineEnd + 1

		if len(line) == 0 {
			if len(p.current.Data) > 0 || len(p.current.Event) > 0 || len(p.current.ID) > 0 {
				events = append(events, p.current)
				p.current = SSEEvent{}
			}
			continue
		}

		if line[0] == ':' {
			continue
		}

		var field, value []byte
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx != -1 {
			field = line[:colonIdx]
			value = line[colonIdx+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
		} else {
			field = line
			value = []byte{}
		}

		switch string(field) {
		case "data":
			p.current.Data = append(p.current.Data, value...)
			p.current.Data = append(p.current.Data, '\n')
		case "event":
			p.current.Event = append([]byte(nil), value...)
		case "id":
			p.current.ID = append([]byte(nil), value...)
		}
	}

	for i := range events {
		if len(events[i].Data) > 0 && events[i].Data[len(events[i].Data)-1] == '\n' {
			events[i].Data = events[i].Data[:len(events[i].Data)-1]
		}
	}

	if start > 0 {
		copy(p.buffer, p.buffer[start:])
		p.buffer = p.buffer[:bufLen-start]
	}

	return events, nil
}
