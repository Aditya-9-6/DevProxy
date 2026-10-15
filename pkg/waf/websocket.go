package waf

import (
	"encoding/binary"
	"sync"
)

var wsBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 65536)
	},
}

// FrameState tracks the fragmentation state of a WebSocket stream.
type FrameState struct {
	IsFragmented bool
	Buffer       []byte
}

// DeframeAndScan processes a raw WebSocket frame, validates masking, and scans for signatures.
func DeframeAndScan(data []byte, scanner *SignatureScanner, state *FrameState) ([]byte, bool) {
	if len(data) < 2 {
		return nil, false
	}

	fin := (data[0] & 0x80) != 0
	opcode := data[0] & 0x0F
	masked := (data[1] & 0x80) != 0
	payloadLen := uint64(data[1] & 0x7F)

	offset := 2
	if payloadLen == 126 {
		if len(data) < offset+2 {
			return nil, false
		}
		payloadLen = uint64(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	} else if payloadLen == 127 {
		if len(data) < offset+8 {
			return nil, false
		}
		payloadLen = binary.BigEndian.Uint64(data[offset : offset+8])
		offset += 8
	}

	var maskKey []byte
	if masked {
		if len(data) < offset+4 {
			return nil, false
		}
		maskKey = data[offset : offset+4]
		offset += 4
	}

	if len(data) < offset+int(payloadLen) {
		return nil, false
	}

	payload := data[offset : offset+int(payloadLen)]
	var unmasked []byte

	if masked {
		buf := wsBufferPool.Get().([]byte)
		if int(payloadLen) > len(buf) {
			unmasked = make([]byte, payloadLen)
		} else {
			unmasked = buf[:payloadLen]
		}
		copy(unmasked, payload)
		applyMask(unmasked, maskKey)

		// Make a safe persistent copy since pool buffer is reused
		result := make([]byte, payloadLen)
		copy(result, unmasked)
		wsBufferPool.Put(buf)
		unmasked = result
	} else {
		result := make([]byte, len(payload))
		copy(result, payload)
		unmasked = result
	}

	if opcode != 0x00 {
		state.IsFragmented = !fin
	}

	if opcode == 0x01 || opcode == 0x02 || state.IsFragmented {
		scanner.Scan(unmasked)
	}

	return unmasked, fin
}

func applyMask(payload, mask []byte) {
	dataLen := len(payload)
	for i := 0; i < dataLen; i++ {
		payload[i] ^= mask[i%4]
	}
}
