package waf

import (
	"bytes"
	"errors"
)

var (
	ErrInvalidChunkSize = errors.New("invalid chunk size format")
	ErrSmugglingVector  = errors.New("smuggling vector detected in chunk extension")
)

// ChunkInspector performs O(1) validation of chunked-transfer encoding.
type ChunkInspector struct {
	// bitmask for fast validation of hex characters
	validHexMask [256]bool
}

func NewChunkInspector() *ChunkInspector {
	ci := &ChunkInspector{}
	for i := '0'; i <= '9'; i++ {
		ci.validHexMask[i] = true
	}
	for i := 'a'; i <= 'f'; i++ {
		ci.validHexMask[i] = true
	}
	for i := 'A'; i <= 'F'; i++ {
		ci.validHexMask[i] = true
	}
	return ci
}

// InspectChunkHeader validates the chunk size line for smuggling vectors.
func (ci *ChunkInspector) InspectChunkHeader(line []byte) error {
	line = bytes.TrimSpace(line)
	semicolon := bytes.IndexByte(line, ';')
	sizePart := line
	if semicolon != -1 {
		sizePart = line[:semicolon]
		// Check for smuggling vectors in extensions
		if bytes.Contains(line[semicolon:], []byte("Transfer-Encoding")) {
			return ErrSmugglingVector
		}
	}
	for _, b := range sizePart {
		if !ci.validHexMask[b] {
			return ErrInvalidChunkSize
		}
	}
	return nil
}
