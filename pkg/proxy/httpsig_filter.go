package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security/httpsig"
)

// HTTPSigFilter represents a proxy filter for verifying signatures.
type HTTPSigFilter struct {
	verifier *httpsig.Verifier
}

// NewHTTPSigFilter creates a new filter.
func NewHTTPSigFilter(v *httpsig.Verifier) *HTTPSigFilter {
	return &HTTPSigFilter{verifier: v}
}
