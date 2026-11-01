package waf

import (
	"errors"
)

var (
	ErrHeaderInjection = errors.New("waf: header injection or request splitting detected (CRLF)")
	ErrInvalidHeader   = errors.New("waf: invalid header characters detected")
)

// Sanitizer inspects and cleans HTTP headers against injection and splitting attacks.
type Sanitizer struct {
	normalizer *Normalizer
}

// NewSanitizer instantiates a new WAF Header Sanitizer.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		normalizer: NewNormalizer(),
	}
}

// SanitizeValue checks and cleans header values, stripping control characters or returning errors.
func (s *Sanitizer) SanitizeValue(val []byte) ([]byte, error) {
	if FastContainsCRLF(val) {
		return nil, ErrHeaderInjection
	}

	// Strip null bytes or other dangerous control characters if needed
	for _, b := range val {
		if b < 32 && b != '\t' {
			return nil, ErrInvalidHeader
		}
	}

	return val, nil
}

// SanitizeHeaderTuple validates and normalizes a key-value pair.
func (s *Sanitizer) SanitizeHeaderTuple(key, val []byte, scratch []byte) ([]byte, []byte, error) {
	normKey := s.normalizer.NormalizeKey(key, scratch)

	cleanVal, err := s.SanitizeValue(val)
	if err != nil {
		return nil, nil, err
	}

	return normKey, cleanVal, nil
}
