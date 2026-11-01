package http2

import (
	"testing"
)

func TestContinuationGuard(t *testing.T) {
	guard := NewContinuationGuard(100)

	if !guard.CheckChunk(50) {
		valErr(t, "expected chunk to pass")
	}

	if guard.CheckChunk(60) {
		valErr(t, "expected chunk to fail limit")
	}
}

func valErr(t *testing.T, msg string) {
	t.Helper()
	t.Error(msg)
}
