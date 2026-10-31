package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"
)

func TestExtractSPIFFEID(t *testing.T) {
	u, _ := url.Parse("spiffe://example.org/ns/default/sa/test")
	cert := &x509.Certificate{
		URIs: []*url.URL{u},
	}
	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	id, err := ExtractSPIFFEID(state)
	if err != nil {
			t.Fatalf("unexpected error: %v", err)
	}
	if id.TrustDomain != "example.org" {
		t.Errorf("expected trust domain example.org, got %s", id.TrustDomain)
	}
}
