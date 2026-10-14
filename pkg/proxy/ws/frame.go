package ws

import (
	"encoding/binary"
	"errors"
)

type Frame struct {
	Fin     bool
	Opcode  byte
	Masked  bool
	Payload []byte
}

func Mask(payload []byte, maskKey []byte) {
	for i := 0; i < len(payload); i++ {
		payload[i] ^= maskKey[i%4]
	}
}

func ParseFrame(data []byte) (*Frame, int, error) {
	if len(data) < 2 {
		return nil, 0, errors.New("frame too short")
	}
	fin := (data[0] & 0x80) != 0
	opcode := data[0] & 0x0F
	masked := (data[1] & 0x80) != 0
	payloadLen := int(data[1] & 0x7F)
	offset := 2
	if payloadLen == 126 {
		if len(data) < 4 {
			return nil, 0, errors.New("short length")
		}
		payloadLen = int(binary.BigEndian.Uint16(data[2:4]))
		offset = 4
	} else if payloadLen == 127 {
		if len(data) < 10 {
			return nil, 0, errors.New("short length")
		}
		payloadLen = int(binary.BigEndian.Uint64(data[2:10]))
		offset = 10
	}
	var maskKey []byte
	if masked {
		if len(data) < offset+4 {
			return nil, 0, errors.New("short mask")
		}
		maskKey = data[offset : offset+4]
		offset += 4
	}
	if len(data) < offset+payloadLen {
		return nil, 0, errors.New("short payload")
	}
	payload := make([]byte, payloadLen)
	copy(payload, data[offset:offset+payloadLen])
	if masked {
		Mask(payload, maskKey)
	}
	return &Frame{Fin: fin, Opcode: opcode, Masked: masked, Payload: payload}, offset + payloadLen, nil
}
