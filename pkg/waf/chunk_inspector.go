package waf

import (
	"bytes"
)

// ChunkInspector inspects HTTP request/response chunks for malicious payloads.
type ChunkInspector struct {
	signatures [][]byte
}

// NewChunkInspector creates a new ChunkInspector with given signatures.
func NewChunkInspector(signatures [][]byte) *ChunkInspector {
	return &ChunkInspector{
		signatures: signatures,
	}
}

// InspectChunk checks if a chunk contains any of the known attack signatures.
func (ci *ChunkInspector) InspectChunk(chunk []byte) bool {
	for _, sig := range ci.signatures {
		if bytes.Contains(chunk, sig) {
			return true
		}
	}
	return false
}
