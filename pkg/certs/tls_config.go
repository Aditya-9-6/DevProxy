package certs

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"sync"
)

// CertificateManager caches leaf certificates by host to avoid redundant signing.
type CertificateManager struct {
	ca    *CertificateAuthority
	cache sync.Map // map[string]*tls.Certificate
}

// NewCertificateManager creates a new CertificateManager.
func NewCertificateManager(ca *CertificateAuthority) *CertificateManager {
	return &CertificateManager{
		ca: ca,
	}
}

// GetTLSConfig returns a tls.Config suitable for server-side TLS termination (TLS bumping).
func (cm *CertificateManager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := hello.ServerName
			if serverName == "" {
				serverName = "localhost"
			}
			return cm.GetOrCreateCertificate(serverName)
		},
		MinVersion: tls.VersionTLS12,
	}
}

// GetOrCreateCertificate returns a cached certificate or generates a new one.
func (cm *CertificateManager) GetOrCreateCertificate(host string) (*tls.Certificate, error) {
	// Strip port
	cleanHost := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		cleanHost = h
	}
	cleanHost = strings.ToLower(cleanHost)

	// Check cache
	if val, ok := cm.cache.Load(cleanHost); ok {
		return val.(*tls.Certificate), nil
	}

	// Mint new cert
	der, privKey, err := cm.ca.MintCertificate(cleanHost)
	if err != nil {
		return nil, fmt.Errorf("failed to mint certificate for %s: %w", cleanHost, err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{der, cm.ca.caCert.Raw},
		PrivateKey:  privKey,
		Leaf:        nil,
	}
	// Parse leaf for quick validation if needed
	if parsed, err := x509.ParseCertificate(der); err == nil {
		cert.Leaf = parsed
	}

	cm.cache.Store(cleanHost, cert)
	return cert, nil
}

// GetCACertificatePEM returns the root CA in PEM format.
func (cm *CertificateManager) GetCACertificatePEM() []byte {
	return cm.ca.GetCACertificatePEM()
}
