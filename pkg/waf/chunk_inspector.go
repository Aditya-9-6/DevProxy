package waf

import (
	"bytes"
	"errors"
)

var (
	ErrInvalidChunkSize = errors.New("invalid chunk size format")
	ErrSmugglingVector  = errors.New("smuggling vector detected in chunk extension")
)

// ChunkInspector performs O(1) validation of chunked-transfer encoding using a cache-aligned bitmask.
type ChunkInspector struct {
	validHexMask [4]uint64
}

func NewChunkInspector() *ChunkInspector {
	ci := &ChunkInspector{}
	for i := 0; i < 256; i++ {
		if (i >= '0' && i <= '9') || (i >= 'a' && i <= 'f') || (i >= 'A' && i <= 'F') {
			ci.validHexMask[i/64] |= (1 << (uint(i) % 64))
		}
	}
	return ci
}

func (ci *ChunkInspector) isHex(b byte) bool {
	return (ci.validHexMask[b/64] & (1 << (uint(b) % 64))) != 0
}

// InspectChunkHeader validates the chunk size line for smuggling vectors.
func (ci *ChunkInspector) InspectChunkHeader(line []byte) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return ErrInvalidChunkSize
	}
	semicolon := bytes.IndexByte(line, ';')
	sizePart := line
	if semicolon != -1 {
		sizePart = line[:semicolon]
		if bytes.Contains(bytes.ToLower(line[semicolon:]), []byte("transfer-encoding")) {
			return ErrSmugglingVector
		}
	}
	for _, b := range sizePart {
		if !ci.isHex(b) {
			return ErrInvalidChunkSize
		}
	}
	return nil
}
