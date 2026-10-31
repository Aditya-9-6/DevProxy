package wscompress

import (
	"sync"
)

// HistoryRing implements a sliding window ring buffer for DEFLATE compression history (up to 32KiB).
type HistoryRing struct {
	buf  []byte
	head int
	size int
}

var historyPool = sync.Pool{
	New: func() any {
		return &HistoryRing{
			buf: make([]byte, 32*1024),
		}
	},
}

// GetHistoryRing retrieves or allocates a 32KiB history ring buffer from sync.Pool.
func GetHistoryRing() *HistoryRing {
	return historyPool.Get().(*HistoryRing)
}

// PutHistoryRing recycles a history ring back to sync.Pool.
func PutHistoryRing(h *HistoryRing) {
	if h == nil {
		return
	}
	h.Reset()
	historyPool.Put(h)
}

// Reset clears ring state without freeing memory.
func (h *HistoryRing) Reset() {
	h.head = 0
	h.size = 0
}

// Write appends bytes to the sliding window.
func (h *HistoryRing) Write(p []byte) {
	for _, b := range p {
		h.buf[h.head] = b
		h.head = (h.head + 1) % len(h.buf)
		if h.size < len(h.buf) {
			h.size++
		}
	}
}
