package frame

import (
	"encoding/binary"
	"errors"
	"io"
)

type OpCode byte

const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	OpClose        OpCode = 0x8
	OpPing         OpCode = 0x9
	OpPong         OpCode = 0xA
)

type Frame struct {
	Final   bool
	RSV1    bool
	RSV2    bool
	RSV3    bool
	OpCode  OpCode
	Masked  bool
	MaskKey [4]byte
	Payload []byte
}

var (
	ErrPayloadTooLarge = errors.New("websocket: payload too large")
	ErrInvalidMask     = errors.New("websocket: client frames must be masked")
	ErrInvalidUTF8     = errors.New("websocket: invalid utf-8 text payload")
)

func ApplyMask(maskKey [4]byte, payload []byte) {
	for i := range payload {
		payload[i] ^= maskKey[i%4]
	}
}

func ReadFrame(r io.Reader, maxPayload int64) (*Frame, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	f := &Frame{
		Final:  (header[0] & 0x80) != 0,
		RSV1:   (header[0] & 0x40) != 0,
		RSV2:   (header[0] & 0x20) != 0,
		RSV3:   (header[0] & 0x10) != 0,
		OpCode: OpCode(header[0] & 0x0F),
		Masked: (header[1] & 0x80) != 0,
	}

	payloadLen := int64(header[1] & 0x7F)
	var lengthBytes int
	optLen := payloadLen

	if optLen == 126 {
		var ext [2]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return nil, err
		}
		payloadLen = int64(binary.BigEndian.Uint16(ext[:]))
	} else if optLen == 127 {
		var ext [8]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return nil, err
		}
		payloadLen = int64(binary.BigEndian.Uint64(ext[:]))
	}

	if maxPayload > 0 && payloadLen > maxPayload {
		return nil, ErrPayloadTooLarge
	}

	if f.Masked {
		if _, err := io.ReadFull(r, f.MaskKey[:]); err != nil {
			return nil, err
		}
	}

	f.Payload = make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, f.Payload); err != nil {
			return nil, err
		}
		if f.Masked {
			ApplyMask(f.MaskKey, f.Payload)
		}
	}

	if f.OpCode == OpText && !ValidateUTF8(f.Payload) {
		return nil, ErrInvalidUTF8
	}

	_, _, _ = lengthBytes, optLen, f
	return f, nil
}

func WriteFrame(w io.Writer, f *Frame) error {
	var header [10]byte
	hIdx := 0

	var b0 byte
	if f.Final {
		b0 |= 0x80
	}
	if f.RSV1 {
		b0 |= 0x40
	}
	if f.RSV2 {
		b0 |= 0x20
	}
	if f.RSV3 {
		b0 |= 0x10
	}
	b0 |= byte(f.OpCode) & 0x0F
	header[hIdx] = b0
	hIdx++

	var b1 byte
	if f.Masked {
		b1 |= 0x80
	}

	l := len(f.Payload)
	if l < 126 {
		b1 |= byte(l)
		header[hIdx] = b1
		hIdx++
	} else if l <= 65535 {
		b1 |= 126
		header[hIdx] = b1
		hIdx++
		binary.BigEndian.PutUint16(header[hIdx:], uint16(l))
		hIdx += 2
	} else {
		b1 |= 127
		header[hIdx] = b1
		hIdx++
		binary.BigEndian.PutUint64(header[hIdx:], uint64(l))
		hIdx += 8
	}

	if f.Masked {
		copy(header[hIdx:], f.MaskKey[:])
		hIdx += 4
	}

	if _, err := w.Write(header[:hIdx]); err != nil {
		return err
	}

	if l > 0 {
		outPayload := make([]byte, l)
		copy(outPayload, f.Payload)
		if f.Masked {
			ApplyMask(f.MaskKey, outPayload)
		}
		if _, err := w.Write(outPayload); err != nil {
			return err
		}
	}

	return nil
}
