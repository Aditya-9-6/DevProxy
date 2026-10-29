package packet

import (
	"testing"
)

func TestHeaderProtectionRemovalError(t *testing.T) {
	key := make([]byte, 5) // Too short
	sample := make([]byte, 16)
	fb := byte(0xc0)
	pn := make([]byte, 4)

	err := RemoveHeaderProtection(key, sample, &fb, pn)
	if err == nil {
		t.Error("expected error for short key, got nil")
	}
}
