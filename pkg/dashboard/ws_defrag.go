package dashboard

import (
	"errors"
	"sync"
	"unicode/utf8"
)

var (
	ErrFragmentBeforeStart = errors.New("websocket: continuation frame before start frame")
	ErrNestedStartFrame    = errors.New("websocket: start frame before previous message finished")
	ErrMessageTooLarge     = errors.New("websocket: message exceeds maximum size")
	ErrInvalidUTF8         = errors.New("websocket: invalid utf-8 in text frame")
	ErrControlFragmented   = errors.New("websocket: control frames cannot be fragmented")
)

// WSDefragmenter reassembles fragmented WebSocket frames into complete messages.
type WSDefragmenter struct {
	maxSize       int
	currentOpcode int
	buffer        []byte
	pool          *sync.Pool
}

// NewWSDefragmenter creates a new defragmenter with a specific max message size.
func NewWSDefragmenter(maxSize int) *WSDefragmenter {
	return &WSDefragmenter{
		maxSize: maxSize,
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 32768) // 32KB initial capacity
			},
		},
	}
}

// ProcessFrame handles a single WebSocket frame and returns a complete WSMessage when the FIN bit is set.
func (d *WSDefragmenter) ProcessFrame(fin bool, opcode int, data []byte) (*WSMessage, error) {
	// Control frames (opcode >= 8) cannot be fragmented and can appear between fragments.
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
		d.buffer = d.pool.Get().([]byte)
	}
	d.buffer = append(d.buffer, data...)

	if !fin {
		return nil, nil
	}

	// Message is complete. We copy the data to a new slice because the buffer will be recycled.
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

// Reset clears the internal state and returns the buffer to the pool.
func (d *WSDefragmenter) Reset() {
	if d.buffer != nil {
		if cap(d.buffer) <= d.maxSize {
			d.pool.Put(d.buffer[:0])
		}
	}
	d.buffer = nil
	d.currentOpcode = 0
}
