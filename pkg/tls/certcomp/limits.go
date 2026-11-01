package certcomp

import (
	"errors"
)

var (
	ErrCompressedSizeExceeded   = errors.New("certcomp: compressed payload exceeds maximum allowed size")
	ErrExpansionRatioExceeded   = errors.New("certcomp: uncompressed size exceeds maximum expansion ratio")
	ErrCertificateCountExceeded = errors.New("certcomp: certificate chain count exceeds limit")
	ErrVectorLengthExceeded     = errors.New("certcomp: certificate vector length exceeds limit")
	ErrHandshakeTimeout         = errors.New("certcomp: handshake or decompression deadline exceeded")
	ErrUnsupportedAlgorithm     = errors.New("certcomp: unsupported certificate compression algorithm")
	ErrMalformedMessage         = errors.New("certcomp: malformed certificate compression wire format")
)

// Limits configures security bounds for certificate compression and decompression.
type Limits struct {
	MaxCompressedSize   int
	MaxUncompressedSize int
	MaxExpansionRatio   float64
	MaxCertificateCount int
	MaxVectorLength     int
}

// DefaultLimits returns secure defaults per RFC 8879 recommendations.
func DefaultLimits() Limits {
	return Limits{
		MaxCompressedSize:   1 << 20,  // 1 MB
		MaxUncompressedSize: 10 << 20, // 10 MB
		MaxExpansionRatio:   20.0,
		MaxCertificateCount: 64,
		MaxVectorLength:     16 << 20, // 16 MB
	}
}