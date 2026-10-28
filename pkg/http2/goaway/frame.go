package goaway

import (
	"encoding/binary"
	"errors"
	"sync"
)

var (
	ErrInvalidFrameLength = errors.New("goaway: invalid frame length")
	ErrReservedBitSet     = errors.New("goaway: reserved bit set in stream identifier")
)

// Frame represents an RFC 9113 GOAWAY frame.
type Frame struct {
	LastStreamID uint32
	ErrorCode    uint32
	DebugData    []byte
}

var framePool = sync.Pool{
	New: func() interface{} {
		return &Frame{
			DebugData: make([]byte, 0, 256),
		}
	},
}

func AcquireFrame() *Frame {
	f := framePool.Get().(*Frame)
	f.LastStreamID = 0
	f.ErrorCode = 0
	f.DebugData = f.DebugData[:0]
	return f
}

func ReleaseFrame(f *Frame) {
	if f != nil {
		f.DebugData = f.DebugData[:0]
		framePool.Put(f)
	}
}

func ParseFrame(payload []byte) (*Frame, error) {
	if len(payload) < 8 {
		return nil, ErrInvalidFrameLength
	}
	
	lastStreamID := binary.BigEndian.Uint32(payload[0:4])
	if (lastStreamID & 0x80000000) != 0 {
		return nil, ErrReservedBitSet
	}
	
	errCode := binary.BigEndian.Uint32(payload[4:8])
	
	debugData := payload[8:]
	f := AcquireFrame()
	f.LastStreamID = lastStreamID
	f.ErrorCode = errCode
	f.DebugData = append(f.DebugData[:0], debugData...)
	return f, nil
}
