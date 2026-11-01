package proxy

import (
	"context"
	"net/http"

	"github.com/gossipmesh/gossipmesh/pkg/security/httpsig"
)

// HTTPSigFilter provides proxy-level request authentication and signature enforcement using GossipMesh native httpsig.
type HTTPSigFilter struct {
	verifier *httpsig.Verifier
}

// NewHTTPSigFilter creates a proxy signature filter.
func NewHTTPSigFilter(verifier *httpsig.Verifier) *HTTPSigFilter {
	return &HTTPSigFilter{
		verifier: verifier,
	}
}

// Authenticate inspects inbound proxy requests and verifies signatures according to RFC 9421.
func (f *HTTPSigFilter) Authenticate(ctx context.Context, req *http.Request, sigHeader string, components []string, params map[string]string) error {
	if f.verifier == nil {
		return nil
	}
	return f.verifier.Verify(req, sigHeader, components, params)
}
