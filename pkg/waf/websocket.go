package waf

import (
	"sync"
	"sync/atomic"
)

var wsBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

// FrameState tracks the fragmentation state of a WebSocket stream.
type FrameState struct {
	Fragmented atomic.Bool
	PayloadLen atomic.Uint64
}

// DeframeAndScan processes a raw WebSocket frame, validates masking, and scans for signatures.
func DeframeAndScan(data []byte, scanner *SignatureScanner) ([]byte, bool) {
	if len(data) < 2 {
		return nil, false
	}

	fin := (data[0] & 0x80) != 0
	opcode := data[0] & 0x0F
	masked := (data[1] & 0x80) != 0
	payloadLen := uint64(data[1] & 0x7F)

	offset := 2
	if payloadLen == 126 {
		offset += 2
	} else if payloadLen == 127 {
		offset += 8
	}

	var maskKey []byte
	if masked {
		maskKey = data[offset : offset+4]
		offset += 4
	}

	if len(data) < offset+int(payloadLen) {
		return nil, false
	}

	payload := data[offset : offset+int(payloadLen)]
	if masked {
		for i := 0; i < len(payload); i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	if opcode == 0x01 || opcode == 0x02 {
		scanner.Scan(payload)
	}

	return payload, fin
}
