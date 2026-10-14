package sse

import (
	"bytes"
)

type SSEParser struct {
	buffer []byte
}

func NewSSEParser() *SSEParser {
	return &SSEParser{buffer: make([]byte, 0, 4096)}
}

func (p *SSEParser) ParseChunk(chunk []byte) (events [][]byte, err error) {
	p.buffer = append(p.buffer, chunk...)
	for {
		idx := bytes.Index(p.buffer, []byte("\n\n"))
		if idx == -1 {
			break
		}
		event := make([]byte, idx)
		copy(event, p.buffer[:idx])
		events = append(events, event)
		p.buffer = p.buffer[idx+2:]
	}
	return events, nil
}
