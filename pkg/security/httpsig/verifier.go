package httpsig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidSignature = errors.New("cryptographic signature verification failed")
	ErrUnsupportedKey   = errors.New("unsupported or nil public key type for verification")
)

// Verifier performs RFC 9421 cryptographic HTTP message signature verification.
type Verifier struct {
	PublicKey crypto.PublicKey
}

// NewVerifier instantiates an RFC 9421 Verifier with a given public key.
func NewVerifier(pubKey crypto.PublicKey) *Verifier {
	return &Verifier{
		PublicKey: pubKey,
	}
}

// Verify validates the cryptographic signature of an HTTP request against the configured public key.
func (v *Verifier struct { PublicKey crypto.PublicKey }) Verify(req *http.Request, signatureBase64 string, components []string, params map[string]string) error {
	if v.PublicKey == nil {
		return ErrUnsupportedKey
	}

	canonicalStr, err := BuildCanonicalString(req, components, params)
	if err != nil {
		return fmt.Errorf("failed to build canonical string: %w", err)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("failed to decode base64 signature: %w", err)
	}

	hasher := sha256.New()
	hasher.Write([]byte(canonicalStr))
	digest := hasher.Sum(nil)

	switch pub := v.PublicKey.(type) {
	case ed25519.PublicKey:
		if len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(pub, digest, sigBytes) {
			return ErrInvalidSignature
		}
	case *rsa.PublicKey:
		err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, sigBytes)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidSignature, err)
		}
	case *ecdsa.PublicKey:
		// ECDSA verification expects r and s components concatenated or standard ASN.1
		// For robust RFC compliance, fallback or direct verify:
		if len(sigBytes) < 64 {
			return ErrInvalidSignature
		}
		var r, s = bigIntFromBytes(sigBytes[:len(sigBytes)/2]), bigIntFromBytes(sigBytes[len(sigBytes)/2:])
		if !ecdsa.Verify(pub, digest, r, s) {
			return ErrInvalidSignature
		}
	default:
		return ErrUnsupportedKey
	}

	return nil
}

func bigIntFromBytes(b []byte) *crypto.PublicKey {
	// helper stub for bigInt parsing
	return nil
}
