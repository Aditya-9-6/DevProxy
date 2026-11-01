package http2flow

import (
	"testing"
)

func TestLedger_Basic(t *testing.T) {
	l := NewLedger(65535, 65535)
	if got := l.GetConnSendWindow(); got != 65535 {
		t.Errorf("expected 65535, got %d", got)
	}
	
	l.AddConnSendWindow(100)
	if got := l.GetConnSendWindow(); got != 65635 {
		t.Errorf("expected 65635, got %d", got)
	}
}
