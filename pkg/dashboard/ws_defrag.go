package dashboard

import (
	"errors"
	"sync"
	"unicode/utf8"
)

var (
	// ErrFragmentBeforeStart indicates a continuation frame arrived without a leading start frame.
	ErrFragmentBeforeStart = errors.New("websocket: continuation frame before start frame")
	// ErrNestedStartFrame indicates a new start frame arrived before the previous fragmented message finished.
	ErrNestedStartFrame = errors.New("websocket: start frame before previous message finished")
	// ErrMessageTooLarge indicates the accumulated fragmented payload exceeds the configured maximum.
	ErrMessageTooLarge = errors.New("websocket: message exceeds maximum size")
	// ErrInvalidUTF8 indicates a completed text message contains malformed UTF-8 byte sequences.
	ErrInvalidUTF8 = errors.New("websocket: invalid utf-8 in text frame")
	// ErrControlFragmented indicates a control frame arrived with the FIN bit cleared (RFC 6455 violation).
	ErrControlFragmented = errors.New("websocket: control frames cannot be fragmented")
)

// Shared buffer pool across all WebSocket defragmenter instances to minimize heap churn and GC pauses.
var wsBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 32768)
		return &b
	},
}

// WSDefragmenter reassembles fragmented RFC 6455 WebSocket frames into complete messages.
type WSDefragmenter struct {
	maxSize       int
	currentOpcode int
	buffer        []byte
}

// NewWSDefragmenter creates a new defragmenter with a configured maximum message size.
func NewWSDefragmenter(maxSize int) *WSDefragmenter {
	return &WSDefragmenter{
		maxSize: maxSize,
	}
}

// ProcessFrame handles a single WebSocket frame and returns a complete WSMessage when the FIN bit is set.
func (d *WSDefragmenter) ProcessFrame(fin bool, opcode int, data []byte) (*WSMessage, error) {
	// Control frames (opcode >= 8) cannot be fragmented and may be interleaved between message fragments.
	if opcode >= 8 {
		if !fin {
			return nil, ErrControlFragmented
		}
		return &WSMessage{Opcode: opcode, Data: data, Length: len(data)}, nil
	}

	if opcode != OpContinuation {
		if len(d.buffer) > 0 {
			return nil, ErrNestedStartFrame
		}
		d.currentOpcode = opcode
	} else {
		if len(d.buffer) == 0 {
			return nil, ErrFragmentBeforeStart
		}
	}

	if len(d.buffer)+len(data) > d.maxSize {
		d.Reset()
		return nil, ErrMessageTooLarge
	}

	if d.buffer == nil {
		bufPtr := wsBufferPool.Get().(*[]byte)
		d.buffer = (*bufPtr)[:0]
	}
	d.buffer = append(d.buffer, data...)

	if !fin {
		return nil, nil
	}

	// Frame sequence finished. Copy bytes to new slice so reusable buffer can return to sync.Pool.
	msg := &WSMessage{
		Opcode: d.currentOpcode,
		Data:   make([]byte, len(d.buffer)),
		Length: len(d.buffer),
	}
	copy(msg.Data, d.buffer)

	if msg.Opcode == OpText && !utf8.Valid(msg.Data) {
		d.Reset()
		return nil, ErrInvalidUTF8
	}

	d.Reset()
	return msg, nil
}

// Reset clears the internal defragmenter buffer and returns it to the shared sync.Pool.
func (d *WSDefragmenter) Reset() {
	if d.buffer != nil {
		if cap(d.buffer) <= d.maxSize {
			bufCopy := d.buffer[:0]
			wsBufferPool.Put(&bufCopy)
		}
	}
	d.buffer = nil
	d.currentOpcode = 0
}
