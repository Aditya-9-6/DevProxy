package certs

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestCertificateAuthority(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "test-ca.crt")
	keyPath := filepath.Join(tempDir, "test-ca.key")

	ca, err := NewCertificateAuthority(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("CA cert file was not created: %v", err)
	}

	pem := ca.GetCACertificatePEM()
	if len(pem) == 0 {
		t.Fatalf("CA PEM is empty")
	}

	// Test dynamic leaf cert minting
	leafDER, leafKey, err := ca.MintCertificate("api.stripe.com")
	if err != nil {
		t.Fatalf("Failed to mint leaf certificate: %v", err)
	}
	if leafKey == nil || len(leafDER) == 0 {
		t.Fatalf("Invalid leaf cert or key")
	}

	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("Failed to parse minted leaf cert: %v", err)
	}

	// Verify certificate chains back to our CA
	roots := x509.NewCertPool()
	roots.AddCert(ca.caCert)

	opts := x509.VerifyOptions{
		DNSName: "api.stripe.com",
		Roots:   roots,
	}

	if _, err := leafCert.Verify(opts); err != nil {
		t.Fatalf("Minted certificate failed verification against CA: %v", err)
	}
}
