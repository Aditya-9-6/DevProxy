package certs

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CertificateAuthority handles generation of CA and dynamic minting of leaf certificates for TLS bumping.
type CertificateAuthority struct {
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	certCache sync.Map // map[string]*tls.Certificate
}

// NewCertificateAuthority initializes or loads a CA from the provided paths or default config directory.
func NewCertificateAuthority(certPath, keyPath string) (*CertificateAuthority, error) {
	if certPath == "" || keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		caDir := filepath.Join(home, ".devproxy")
		_ = os.MkdirAll(caDir, 0700)
		certPath = filepath.Join(caDir, "devproxy-ca.crt")
		keyPath = filepath.Join(caDir, "devproxy-ca.key")
	}

	ca := &CertificateAuthority{}

	// Check if already exists
	if fileExists(certPath) && fileExists(keyPath) {
		if err := ca.loadCA(certPath, keyPath); err == nil {
			return ca, nil
		}
	}

	// Generate new Root CA
	if err := ca.generateCA(certPath, keyPath); err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	return ca, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (ca *CertificateAuthority) loadCA(certPath, keyPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8
		k, err2 := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return fmt.Errorf("unable to parse private key: %v / %v", err, err2)
		}
		var ok bool
		key, ok = k.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("expected RSA private key for CA")
		}
	}

	ca.caCert = cert
	ca.caKey = key
	return nil
}

func (ca *CertificateAuthority) generateCA(certPath, keyPath string) error {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"DevProxy Local Authority"},
			CommonName:    "DevProxy Root CA",
			Country:       []string{"US"},
			Province:      []string{"Local"},
			Locality:      []string{"Development"},
			StreetAddress: []string{"Localhost"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return err
	}

	// Save to disk
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}); err != nil {
		return err
	}

	ca.caCert = cert
	ca.caKey = privKey
	return nil
}

// GetCACertificatePEM returns the Root CA certificate in PEM format.
func (ca *CertificateAuthority) GetCACertificatePEM() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.caCert.Raw,
	})
}

// MintCertificate dynamically signs a leaf certificate for the given hostname using fast ECDSA P-256.
func (ca *CertificateAuthority) MintCertificate(rawHost string) (certDER []byte, keyDER crypto.Signer, err error) {
	// Strip port if present
	host := rawHost
	if h, _, err := net.SplitHostPort(rawHost); err == nil {
		host = h
	}

	// Generate fast ECDSA P-256 key for the leaf
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate leaf key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"DevProxy Intercepted"},
		},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
		// If domain has subdomain, also support wildcard
		parts := strings.Split(host, ".")
		if len(parts) > 2 {
			wildcard := "*." + strings.Join(parts[1:], ".")
			template.DNSNames = append(template.DNSNames, wildcard)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, ca.caCert, &leafKey.PublicKey, ca.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign leaf cert: %w", err)
	}

	return der, leafKey, nil
}
