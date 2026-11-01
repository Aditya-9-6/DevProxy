package tls13

import (
	"crypto"
	"testing"
)

func TestFinished_Verify(t *testing.T) {
	hashFunc := crypto.SHA256
	baseSecret := make([]byte, 32)
	transcriptSum := make([]byte, 32)

	finKey := FinishedKey(hashFunc, baseSecret)
	verifyData := ComputeFinished(hashFunc, finKey, transcriptSum)

	if !VerifyFinished(hashFunc, finKey, transcriptSum, verifyData) {
		t.Errorf("failed to verify valid finished message")
	}

	badVerifyData := make([]byte, len(verifyData))
	copy(badVerifyData, verifyData)
	badVerifyData[0] ^= 0xFF

	if VerifyFinished(hashFunc, finKey, transcriptSum, badVerifyData) {
		t.Errorf("falsely verified invalid finished message")
	}
}
