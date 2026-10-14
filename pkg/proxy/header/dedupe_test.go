package header

import (
	"net/http"
	"testing"
)

func TestHeaderDeduplication(t *testing.T) {
	sm := NewStateMachine()
	h := http.Header{"Content-Type": []string{"application/json"}}
	hash := ComputeHeaderHash(h)

	b1 := sm.GetOrStore(hash, []byte("application/json"))
	b2 := sm.GetOrStore(hash, []byte("application/json"))

	if b1 != b2 {
		t.Errorf("Expected same pointer for identical header hash, got %p and %p", b1, b2)
	}
}
