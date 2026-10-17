package replay

import (
	"context"
	"io"
	"sync/atomic"
)

// CacheLinePad prevents false sharing across concurrent CPU cores.
type CacheLinePad [56]byte

// ReplayBufferChunk represents an atomic cache-aligned node in the ring buffer.
type ReplayBufferChunk struct {
	_      CacheLinePad
	Data   []byte
	Length atomic.Int64
	Next   *ReplayBufferChunk
	_      CacheLinePad
}

// ReplayRingBuffer provides a lock-free, cache-aligned ring buffer of atomic pointers
// for zero-allocation HTTP request duplication and replay.
type ReplayRingBuffer struct {
	capacity int
	chunks   []*ReplayBufferChunk
	head     atomic.Uint64
	tail     atomic.Uint64
	pool     *BufferPool
}

// NewReplayRingBuffer creates a fixed-size, cache-aligned atomic ring buffer.
func NewReplayRingBuffer(capacity int, chunkSize int) *ReplayRingBuffer {
	if capacity <= 0 {
		capacity = 1024
	}
	if chunkSize <= 0 {
		chunkSize = 4096
	}

	chunks := make([]*ReplayBufferChunk, capacity)
	for i := range chunks {
		chunks[i] = &ReplayBufferChunk{
			Data: make([]byte, chunkSize),
		}
	}

	// Link chunks together in a ring
	for i := 0; i < capacity; i++ {
		chunks[i].Next = chunks[(i+1)%capacity]
	}

	return &ReplayRingBuffer{
		capacity: capacity,
		chunks:   chunks,
		pool:     NewBufferPool(chunkSize),
	}
}

// WriteChunk writes data into the ring buffer atomically without heap allocation.
func (r *ReplayRingBuffer) WriteChunk(p []byte) int {
	if len(p) == 0 {
		return 0
	}

	headIdx := r.head.Add(1) - 1
	idx := int(headIdx) % r.capacity
	chunk := r.chunks[idx]

	copySize := len(p)
	if copySize > len(chunk.Data) {
		copySize = len(chunk.Data)
	}

	copy(chunk.Data, p[:copySize])
	chunk.Length.Store(int64(copySize))

	return copySize
}

// ReplayReader wraps the ring buffer chunks into an io.Reader for forensic consumers.
type ReplayReader struct {
	ring       *ReplayRingBuffer
	currIdx    int
	offset     int
	maxChunks  int
	readChunks int
}

// NewReplayReader creates a stream reader over the replay buffer.
func NewReplayReader(ring *ReplayRingBuffer) *ReplayReader {
	head := ring.head.Load()
	startIdx := 0
	if head > uint64(ring.capacity) {
		startIdx = int(head-uint64(ring.capacity)) % ring.capacity
	}

	max := ring.capacity
	if head < uint64(max) {
		max = int(head)
	}

	return &ReplayReader{
		ring:       ring,
		currIdx:    startIdx,
		offset:     0,
		maxChunks:  max,
		readChunks: 0,
	}
}

// Read implements the io.Reader interface with zero allocation.
func (r *ReplayReader) Read(p []byte) (n int, err error) {
	if r.readChunks >= r.maxChunks {
		return 0, io.EOF
	}

	chunk := r.ring.chunks[r.currIdx]
	chunkLen := int(chunk.Length.Load())

	if r.offset >= chunkLen {
		r.readChunks++
		r.currIdx = (r.currIdx + 1) % r.ring.capacity
		r.offset = 0
		if r.readChunks >= r.maxChunks {
			return 0, io.EOF
		}
		chunk = r.ring.chunks[r.currIdx]
		chunkLen = int(chunk.Length.Load())
	}

	available := chunkLen - r.offset
	if available <= 0 {
		r.readChunks++
		r.currIdx = (r.currIdx + 1) % r.ring.capacity
		r.offset = 0
		return 0, nil
	}

	n = len(p)
	if n > available {
		n = available
	}

	copy(p[:n], chunk.Data[r.offset:r.offset+n])
	r.offset += n

	return n, nil
}

// DuplicateAndReplay wraps an incoming io.Reader, duplicating its payload on the fly into the ReplayRingBuffer
// while allowing downstream consumers to read concurrently with zero heap allocation.
func DuplicateAndReplay(ctx context.Context, src io.Reader, ring *ReplayRingBuffer) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		buf := ring.pool.Get()
		defer ring.pool.Put(buf)

		for {
			select {
			case <-ctx.Done():
				_ = pw.CloseWithError(ctx.Err())
				return
			default:
			}

			n, err := src.Read(buf)
			if n > 0 {
				ring.WriteChunk(buf[:n])
				if _, wErr := pw.Write(buf[:n]); wErr != nil {
					return
				}
			}

			if err != nil {
				if err != io.EOF {
					_ = pw.CloseWithError(err)
				}
				return
			}
		}
	}()

	return pr, nil
}
