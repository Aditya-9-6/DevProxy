package tls13

import (
	"crypto"
	"bytes"
	"testing"
)

func TestTranscriptHash_Basic(t *testing.T) {
	th := NewTranscriptHash(crypto.SHA256)
	if th.Size() != 32 {
		t.Errorf("expected size 32, got %d", th.Size())
	}

	_, _ = th.Write([]byte("hello"))
	sum1 := th.Sum()
	if len(sum1) != 32 {
		t.Errorf("expected sum length 32, got %d", len(sum1))
	}

	th.Reset()
	sum2 := th.Sum()
	// After reset, sum should be empty hash
	emptyTh := NewTranscriptHash(crypto.SHA256)
	resetSum := emptyTh.Sum()
	if !bytes.Equal(sum2, resetSum) {
		t.Errorf("reset transcript does not match fresh transcript")
	}
}
