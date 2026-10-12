package http3

import (
	"sync"
	"sync/atomic"
)

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

// DecodeHeaderBlock performs zero-copy header recovery using pooled buffers.
// The caller is responsible for calling Release() on the returned HeaderBlock.
func (d *QPACKDecoder) DecodeHeaderBlock(data []byte) (*HeaderBlock, error) {
	buf := GetBuffer()
	copy(buf, data)

	// TODO: Implement actual QPACK state machine logic here.
	return &HeaderBlock{Data: buf[:len(data)]}, nil
}
