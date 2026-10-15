package h2

const MaxHeaderBlockSize = 1024 * 1024 // 1MB limit per header block

// StreamTracker manages state for up to 65536 concurrent streams using a fixed-size array.
type StreamTracker struct {
	streams [65536]*StreamState
}

func NewStreamTracker() *StreamTracker {
	t := &StreamTracker{}
	for i := 0; i < 65536; i++ {
		t.streams[i] = &StreamState{ID: uint32(i)}
	}
	return t
}

func (t *StreamTracker) Reset(streamID uint32) {
	idx := streamID % 65536
	t.streams[idx].HeaderBlockSize.Store(0)
	t.streams[idx].ContinuationCount.Store(0)
}

func (t *StreamTracker) Get(streamID uint32) *StreamState {
	return t.streams[streamID%65536]
}
