package http2header

import (
	"testing"
)

func TestPressureAccountant(t *testing.T) {
	pa := NewPressureAccountant(100)

	if !pa.AddBytes(50) {
		TErrorf(t, "expected 50 bytes to be allowed")
	}

	if pa.AddBytes(60) {
		TErrorf(t, "expected 110 total bytes to exceed 100 limit")
	}

	pa.Reset()
	if pa.Total() != 0 {
		TErrorf(t, "expected total to be reset to 0")
	}
}

func TErrorf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Errorf(format, args...)
}
