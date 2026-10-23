package multipart

import (
	"bytes"
)

// BoundaryMatcher handles zero-allocation rolling window boundary detection.
type BoundaryMatcher struct {
	boundary []byte
	full     []byte
	close    []byte
}

func NewBoundaryMatcher(boundary string) *BoundaryMatcher {
	b := []byte(boundary)
	full := append([]byte("--"), b...)
	close := append(append([]byte("--"), b...), []byte("--")...)
	return &BoundaryMatcher{
		boundary: b,
		full:     full,
		close:    close,
	}
}

func (bm *BoundaryMatcher) MatchFull(chunk []byte) bool {
	return bytes.Contains(chunk, bm.full)
}

func (bm *BoundaryMatcher) MatchClose(chunk []byte) bool {
	return bytes.Contains(chunk, bm.close)
}
