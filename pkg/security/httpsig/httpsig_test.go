package httpsig

import (
	"testing"
)

func TestKeyringAndVerifier(t *testing.T) {
	kr := NewKeyring()
	kr.AddKey("key-1", []byte("dummy-key"))

	v := NewVerifier(kr)
	err := v.Verify("sig", "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
