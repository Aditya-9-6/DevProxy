package websocket

import (
	"testing"
)

func TestBridge(t *testing.T) {
	b := NewBridge()
	if b == nil {
		t.Fatalf("nil bridge")
	}
}