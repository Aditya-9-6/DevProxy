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
		payloadLen = uint64(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2
	} else if payloadLen == 127 {
		payloadLen = binary.BigEndian.Uint64(data[offset : offset+8])
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
	var unmasked []byte

	if masked {
		unmasked = wsBufferPool.Get().([]byte)[:payloadLen]
		copy(unmasked, payload)
		applyMask(unmasked, maskKey)
	} else {
		unmasked = payload
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
	maskUint32 := binary.LittleEndian.Uint32(mask)
	i := 0
	for ; i+8 <= len(payload); i += 8 {
		val := binary.LittleEndian.Uint64(payload[i:])
		mask8 := uint64(maskUint32) | (uint64(maskUint32) << 32)
		binary.LittleEndian.PutUint64(payload[i:], val^mask8)
	}
	for ; i < len(payload); i++ {
		payload[i] ^= mask[i%4]
	}
}
