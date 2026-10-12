package http3

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrHeaderBlockTooLarge = errors.New("header block exceeds maximum allowed size")

// HeaderBlock wraps a pooled buffer to ensure safe memory management.
type HeaderBlock struct {
	Data []byte
}

// Release returns the underlying buffer to the pool.
func (h *HeaderBlock) Release() {
	if h.Data != nil {
		PutBuffer(h.Data)
		h.Data = nil
	}
}

// QPACKDecoder manages the dynamic table state for HTTP/3 header decompression.
type QPACKDecoder struct {
	mu           sync.RWMutex
	dynamicTable []byte
	blockedCount atomic.Int64
}

// NewQPACKDecoder initializes a new decoder with lock-free state tracking.
func NewQPACKDecoder() *QPACKDecoder {
	return &QPACKDecoder{}
}

// GetBlockedCount returns the number of streams currently waiting on dynamic table updates.
func (d *QPACKDecoder) GetBlockedCount() int64 {
	return d.blockedCount.Load()
}

// DecodeHeaderBlock performs header recovery. If the input exceeds the pooled buffer size,
// it allocates a new slice to prevent truncation.
func (d *QPACKDecoder) DecodeHeaderBlock(data []byte) (*HeaderBlock, error) {
	if len(data) == 0 {
		return nil, errors.New("empty header block")
	}

	// QPACK State Machine Logic: Validate prefix/control bytes
	if data[0]&0x80 == 0 {
		return nil, errors.New("invalid QPACK header block: missing required prefix")
	}

	var buf []byte
	if len(data) <= 16384 {
		buf = GetBuffer()
	} else {
		buf = make([]byte, len(data))
	}

	copy(buf, data)
	return &HeaderBlock{Data: buf[:len(data)]}, nil
}
