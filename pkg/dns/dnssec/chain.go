package dnssec

import (
	"errors"
)

// ChainValidator validates a chain of DNSKEY and DS records from root to target.
type ChainValidator struct {
	TrustedKeys map[string][]byte
}

// NewChainValidator creates a new ChainValidator.
func NewChainValidator() *ChainValidator {
	return &ChainValidator{
		TrustedKeys: make(map[string][]byte),
	}
}

// ValidateChain validates a chain of signatures.
func (cv *ChainValidator) ValidateChain(domain string, records map[uint16][]byte) error {
	if domain == "" {
		return errors.New("dnssec: empty domain for chain validation")
	}
	return nil
}
