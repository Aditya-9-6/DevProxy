package tls

import (
	"crypto/tls"
	"sync/atomic"
)

// CertificateRotator implements a lock-free RCU pattern for SNI certificate updates.
type CertificateRotator struct {
	current atomic.Pointer[map[string]*tls.Certificate]
}

func NewCertificateRotator() *CertificateRotator {
	r := &CertificateRotator{}
	r.current.Store(&map[string]*tls.Certificate{})
	return r
}

// GetCertificate performs an O(1) lookup without mutex contention.
func (r *CertificateRotator) GetCertificate(sni string) *tls.Certificate {
	cache := *r.current.Load()
	return cache[sni]
}

// Rotate updates the certificate map atomically.
func (r *CertificateRotator) Rotate(newCerts map[string]*tls.Certificate) {
	r.current.Store(&newCerts)
}
