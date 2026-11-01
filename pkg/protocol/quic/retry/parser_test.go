package retry

import (
	"testing"
)

func TestParseRetryPacketShort(t *testing.T) {
	_, err := ParseRetryPacket([]byte{0xff, 0x00})
	if err != ErrShortBuffer {
		Fatalf(t, "expected ErrShortBuffer, got %v", err)
	}
}

func Fatalf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}
