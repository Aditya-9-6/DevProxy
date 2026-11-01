package dnssec

import (
	"testing"
)

func TestChainValidator(t *testing.T) {
	cv := NewChainValidator()
	err := cv.ValidateChain("example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
