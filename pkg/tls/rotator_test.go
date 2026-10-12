package tls

import (
	"crypto/tls"
	"testing"
)

func TestCertificateRotator(t *testing.T) {
	rotator := NewCertificateRotator()
	cert := &tls.Certificate{}

	// Test initial state
	if rotator.GetCertificate("example.com") != nil {
		t.Error("Expected nil for unknown SNI")
	}

	// Test rotation
	newMap := map[string]*tls.Certificate{
		"example.com": cert,
	}
	rotator.Rotate(newMap)

	if rotator.GetCertificate("example.com") != cert {
		t.Error("Failed to retrieve rotated certificate")
	}
}
