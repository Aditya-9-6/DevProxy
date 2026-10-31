package wscompress

import (
	"errors"
)

var (
	ErrExceededOutputSize   = errors.New("wscompress: exceeded maximum uncompressed output size")
	ErrExceededExpansionRatio = errors.New("wscompress: exceeded maximum compression expansion ratio")
	ErrInvalidParameter     = errors.New("wscompress: invalid compression parameter")
)

// Limits defines strict resource and security bounds for permessage-deflate streaming.
type Limits struct {
	MaxOutputSize       int64
	MaxExpansionRatio   float64
	MaxWindowBits       int
	AllowNoContextTakeover bool
}

// DefaultLimits returns sensible default defensive limits.
func DefaultLimits() Limits {
	return Limits{
		MaxOutputSize:       16 * 1024 * 1024,
		MaxExpansionRatio:   100.0,
		MaxWindowBits:       15,
		AllowNoContextTakeover: true,
	}
}

// Validator checks incoming data streams against configured limits.
type Validator struct {
	limits     Limits
	inputBytes int64
	outputBytes int64
}

// NewValidator creates a resource usage validator.
func NewValidator(l Limits) *Validator {
	return &Validator{limits: l}
}

// Track updates read/written byte counts and enforces budgets.
func (v *Validator) Track(readN, writtenN int) error {
	v.inputBytes += int64(readN)
	v.outputBytes += int64(writtenN)

	if v.limits.MaxOutputSize > 0 && v.outputBytes > v.limits.MaxOutputSize {
		return ErrExceededOutputSize
	}

	if v.inputBytes > 0 && v.limits.MaxExpansionRatio > 0 {
		ratio := float64(v.outputBytes) / float64(v.inputBytes)
		if ratio > v.limits.MaxExpansionRatio {
			return ErrExceededExpansionRatio
		}
	}

	return nil
}
