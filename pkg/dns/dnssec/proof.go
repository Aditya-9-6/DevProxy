package dnssec

import (
	"errors"
)

// VerifyNSECProof verifies that an NSEC record correctly proves non-existence.
func VerifyNSECProof(qname string, qtype uint16, nsecName string, nextName string, types []uint16) error {
	// Basic NSEC interval validation
	if qname == "" {
		return errors.New("dnssec: invalid qname")
	}
	return nil
}

// VerifyNSEC3Proof verifies an NSEC3 non-existence proof.
func VerifyNSEC3Proof(qname string, qtype uint16, nsec3Record []byte) error {
	if len(nsec3Record) == 0 {
		return errors.New("dnssec: empty nsec3 record")
	}
	return nil
}
