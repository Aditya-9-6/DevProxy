package frame

import (
	"testing"
)

func TestFrameBasic(t *testing.T) {
	if MaxHeaderSize != 14 {
		t.Errorf("expected 14, got %d", MaxHeaderSize)
	}
}