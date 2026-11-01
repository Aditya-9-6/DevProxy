package proxy

import (
	"net/netip"
	"testing"
	"time"
)

func TestQUICAdmissionController(t *testing.T) {
	ctrl, err := NewQUICAdmissionController(1, time.Minute)
	if err != nil {
		t.Fatalf("failed to create admission controller: %v", err)
	}

	if ctrl.ShouldChallenge() {
		t.Errorf("expected no challenge when below max conns")
	}

	ctrl.IncConns()
	if !ctrl.ShouldChallenge() {
		t.Errorf("expected challenge when at max conns")
	}

	ip := netip.MustParseAddr("10.0.0.1")
	dcid := []byte("dcid-test")

	token, err := ctrl.RetryHandler().GenerateRetryToken(ip, dcid)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parsedDCID, err := ctrl.ValidateToken(ip, token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if string(parsedDCID) != string(dcid) {
		t.Errorf("expected DCID %s, got %s", dcid, parsedDCID)
	}
}
