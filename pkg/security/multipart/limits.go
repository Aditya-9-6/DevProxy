package multipart

import (
	"errors"
)

var (
	ErrHeaderTooLarge = errors.New("multipart: header exceeds maximum allowed size")
	ErrPartLimitExceeded = errors.New("multipart: maximum number of parts exceeded")
	ErrLineTooLong = errors.New("multipart: line exceeds maximum allowed length")
	ErrMalformedBoundary = errors.New("multipart: malformed boundary or preamble")
)

type Limits struct {
	MaxHeaderBytes int
	MaxParts       int
	MaxLineLength  int
}

func DefaultLimits() Limits {
	return Limits{
		MaxHeaderBytes: 4096,
		MaxParts:       100,
		MaxLineLength:  8192,
	}
}
