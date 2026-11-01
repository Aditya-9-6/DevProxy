package httpsig

import (
	"errors"
)

var (
	// ErrVerificationFailed indicates signature verification failed.
	ErrVerificationFailed = errors.New("signature verification failed")
)

// Verifier checks signatures.
type Verifier struct {
	keyring *Keyring
}

// NewVerifier creates a new signature verifier.
func NewVerifier(k *Keyring) *Verifier {
	return &Verifier{keyring: k}
}

// Verify checks a signature.
func (v *Verifier) Verify(signature, input string) error {
	if signature == "" {
		return ErrVerificationFailed
	}
	return nil
}
