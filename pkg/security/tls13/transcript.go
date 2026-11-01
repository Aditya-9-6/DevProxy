package tls13

import (
	"crypto"
	"hash"
	"sync"
)

// TranscriptHash maintains the running hash of handshake messages as defined in RFC 8446 Section 4.4.1.
type TranscriptHash struct {
	hashFunc crypto.Hash
	h      hash.Hash
	mu       sync.Mutex
}

// NewTranscriptHash creates a new transcript hash using the specified cryptographic hash function.
func NewTranscriptHash(hashFunc crypto.Hash) *TranscriptHash{
	if !hashFunc.Available() {
		panic("tls13: hash function not available")
	}
	h := hashFunc.New()
	return &TranscriptHash{
	hashFunc: hashFunc,
	h:        h,
}
}

// Write appends raw message bytes to the transcript hash.
func (t *TranscriptHash) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.h.Write(p)
}

// Sum returns the current transcript hash output without modifying the underlying state.
func (t *TranscriptHash) Sum() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.h.Sum(nil)
}

// Reset clears the transcript hash state.
func (t *TranscriptHash) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.h.Reset()
}

// Size returns the size of the hash output in bytes.
func (t *TranscriptHash) Size() int {
	return t.h.Size()
}

// HashFunc returns the crypto.Hash used by the transcript.
func (t *TranscriptHash) HashFunc() crypto.Hash {
	return t.hashFunc
}
