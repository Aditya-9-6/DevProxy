package proxy

import (
	"testing"
	"time"
)

func TestMitigator_ConnectionTracking(t *testing.T) {
	m := NewMitigator()
	m.TrackConnection()
	m.TrackConnection()
	if m.ActiveConnections[0] != 2 {
		t.Errorf("expected 2 active connections, got %d", m.ActiveConnections[0])
	}
	m.ReleaseConnection()
	if m.ActiveConnections[0] != 1 {
		t.Errorf("expected 1 active connection, got %d", m.ActiveConnections[0])
	}
}

func TestMitigator_DeadlineExpiration(t *testing.T) {
	m := NewMitigator()

	// Test no deadline
	if m.IsExpired() {
		t.Error("should not be expired with no deadline")
	}

	// Test future deadline
	m.SetReadDeadline(time.Now().Add(1 * time.Hour))
	if m.IsExpired() {
		t.Error("should not be expired with future deadline")
	}

	// Test past deadline
	m.SetReadDeadline(time.Now().Add(-1 * time.Hour))
	if !m.IsExpired() {
		t.Error("should be expired with past deadline")
	}
}

func TestMitigator_ViolationTracking(t *testing.T) {
	m := NewMitigator()
	m.RecordViolation()
	m.RecordViolation()
	if m.SlowLorisViolations[0] != 2 {
		t.Errorf("expected 2 violations, got %d", m.SlowLorisViolations[0])
	}
}
