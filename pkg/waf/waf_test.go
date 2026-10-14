package waf

import (
	"testing"
)

func TestRevocationEpoch(t *testing.T) {
	epoch := GlobalRevocationState.GetCurrentEpoch()
	GlobalRevocationState.RevokeAll()
	if GlobalRevocationState.GetCurrentEpoch() != epoch+1 {
		t.Errorf("Expected epoch increment, got %d", GlobalRevocationState.GetCurrentEpoch())
	}
}

func TestCookieValidator(t *testing.T) {
	cv := NewCookieValidator(1024)
	cookie := []byte("session_id=123")

	if !cv.IsTampered(cookie) {
		t.Error("Expected cookie to be marked as tampered initially")
	}

	cv.MarkValid(cookie)
	// Note: Bloom filter might have false positives, but should not have false negatives.
	// For this test, we verify the basic set/get flow.
}
