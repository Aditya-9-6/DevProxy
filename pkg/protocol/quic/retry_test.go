package quic

import (
	"net/netip"
	"testing"
	"time"
)

func TestRetryHandlerWorkflow(t *testing.T) {
	rh, err := NewRetryHandler(time.Minute)
	if err != nil {
		t.Fatalf("failed to create retry handler: %v", err)
	}

	ip := netip.MustParseAddr("203.0.113.5")
	dcid := []byte("client-dcid-123")

	token, err := rh.GenerateRetryToken(ip, dcid)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedDCID, err := rh.VerifyAndCheckReplay(ip, token)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	if string(parsedDCID) != string(dcid) {
		t.Errorf("expected DCID %s, got %s", dcid, parsedDCID)
	}

	// Replay attempt should fail
	_, err = rh.VerifyAndCheckReplay(ip, token)
	if err == nil {
		t.Errorf("expected replay attack to be detected and rejected")
	}
}
