package multipart

import (
	"testing"
)

func TestBoundaryMatcher(t *testing.T) {
	bm := NewBoundaryMatcher("abc123xyz")
	if !bm.MatchFull([]byte("--abc123xyz\r\n")) {
		t.Errorf("Expected match full")
	}
	if !bm.MatchClose([]byte("--abc123xyz--\r\n")) {
		t.Errorf("Expected match close")
	}
}
