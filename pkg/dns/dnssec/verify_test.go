package dnssec

import (
	"testing"
)

func TestVerifyRRSIG(t *testing.T) {
	err := VerifyRRSIG(nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil public key")
	}
}
