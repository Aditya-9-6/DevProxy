package hpack

import (
	"errors"
	"strings"
)

// Validator performs zero-allocation validation on HPACK header fields.
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateField checks if a header field violates HTTP/2 HPACK/pseudo-header rules.
func (v *Validator) ValidateField(state *State, name, value string) error {
	isPseudo := strings.HasPrefix(name, ":")

	if isPseudo {
		if !state.LastWasPseudo {
			return errors.New("pseudo-header found after regular header")
		}
		if state.SeenPseudoHeaders[name] {
			return errors.New("duplicate pseudo-header")
		}
		state.SeenPseudoHeaders[name] = true
	} else {
		state.LastWasPseudo = false
	}

	return nil
}
