package hpack

import (
	"errors"
	"fmt"
)

// Sentinel errors for HPACK validation violations.
var (
	ErrPseudoHeaderAfterRegular = errors.New("pseudo-header found after regular header")
	ErrDuplicatePseudoHeader    = errors.New("duplicate pseudo-header")
	ErrEmptyHeaderName          = errors.New("empty header name")
	ErrUppercaseHeaderName      = errors.New("header name must be lowercase per RFC 7541")
	ErrUnknownPseudoHeader      = errors.New("unknown pseudo-header")
)

const (
	maskMethod    = 1 << 0
	maskScheme    = 1 << 1
	maskAuthority = 1 << 2
	maskPath      = 1 << 3
	maskStatus    = 1 << 4
)

// Validator performs zero-allocation validation on HPACK header fields.
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateField checks if a header field violates HTTP/2 HPACK/pseudo-header rules.
func (v *Validator) ValidateField(state *State, name, value string) error {
	if len(name) == 0 {
		return ErrEmptyHeaderName
	}

	isPseudo := name[0] == ':'

	// Check lowercase requirement per RFC 7541
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			return fmt.Errorf("%w: %s", ErrUppercaseHeaderName, name)
		}
	}

	if isPseudo {
		if !state.LastWasPseudo {
			return ErrPseudoHeaderAfterRegular
		}

		var bit uint8
		switch name {
		case ":method":
			bit = maskMethod
		case ":scheme":
			bit = maskScheme
		case ":authority":
			bit = maskAuthority
		case ":path":
			bit = maskPath
		case ":status":
			bit = maskStatus
		default:
			return fmt.Errorf("%w: %s", ErrUnknownPseudoHeader, name)
		}

		if (state.SeenPseudoHeaders & bit) != 0 {
			return ErrDuplicatePseudoHeader
		}
		state.SeenPseudoHeaders |= bit
	} else {
		state.LastWasPseudo = false
	}

	return nil
}
