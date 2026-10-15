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

var ErrMaxBufferExceeded = errors.New("sse: maximum buffer size exceeded")

type SSEEvent struct {
	Data  []byte
	Event []byte
	ID    []byte
}

type SSEParser struct {
	mu        sync.Mutex
	buffer    []byte
	current   SSEEvent
	maxBuffer int
}

func NewSSEParser() *SSEParser {
	return &SSEParser{
		maxBuffer: 1024 * 1024,
		buffer:    sseBufferPool.Get().([]byte),
	}
}

func NewSSEParserWithLimit(maxBuffer int) *SSEParser {
	return &SSEParser{
		maxBuffer: maxBuffer,
		buffer:    sseBufferPool.Get().([]byte),
	}
}

func (p *SSEParser) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.buffer != nil {
		poolBuf := p.buffer[:0]
		sseBufferPool.Put(poolBuf)
		p.buffer = nil
	}
}

func (p *SSEParser) ParseChunk(chunk []byte, onEvent func(SSEEvent)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.buffer)+len(chunk) > p.maxBuffer {
		return ErrMaxBufferExceeded
	}

	p.buffer = append(p.buffer, chunk...)
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
				onEvent(p.current)
				p.current = SSEEvent{}
			}
			continue
		}
		if line[0] == ':' {
			continue
		}

		colonIdx := bytes.IndexByte(line, ':')
		field, value := line, []byte{}
		if colonIdx != -1 {
			field = line[:colonIdx]
			value = line[colonIdx+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
		}

		switch string(field) {
		case "data":
			p.current.Data = append(p.current.Data, value...)
			p.current.Data = append(p.current.Data, '\n')
		case "event":
			p.current.Event = value
		case "id":
			p.current.ID = value
		}
	}

	if start > 0 {
		copy(p.buffer, p.buffer[start:])
		p.buffer = p.buffer[:bufLen-start]
	}
	return nil
}
