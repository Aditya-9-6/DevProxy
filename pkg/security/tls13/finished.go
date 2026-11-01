package tls13

import (
	"crypto"
	"crypto/hmac"
	"crypto/subtle"
)

// FinishedKey derives the Finished key from a base traffic secret according to RFC 8446 Section 4.4.4.
func FinishedKey(hashFunc crypto.Hash, baseSecret []byte) []byte {
	return ExpandLabel(hashFunc, baseSecret, "finished", nil, hashFunc.Size())
}

// ComputeFinished computes the Finished verify_data for a given Finished key and transcript hash.
func ComputeFinished(hashFunc crypto.Hash, finishedKey, transcriptSum []byte) []byte {
	h := hmac.New(hashFunc.New, finishedKey)
	h.Write(transcriptSum)
	return h.Sum(nil)
}

// VerifyFinished verifies a received verify_data in constant time against the computed verify_data.
func VerifyFinished(hashFunc crypto.Hash, finishedKey, transcriptSum, receivedVerifyData []byte) bool {
	computed := ComputeFinished(hashFunc, finishedKey, transcriptSum)
	return subtle.ConstantTimeCompare(computed, receivedVerifyData) == 1
}
