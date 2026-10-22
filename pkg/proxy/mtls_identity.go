package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

var (
	// ErrNilRouteLookup indicates a missing routing snapshot provider.
	ErrNilRouteLookup = errors.New("proxy: mTLS route lookup is nil")
	// ErrNoSPIFFEID indicates that a peer certificate has no SPIFFE URI SAN.
	ErrNoSPIFFEID = errors.New("proxy: certificate has no SPIFFE ID URI SAN")
	// ErrInvalidSPIFFEID indicates a malformed or ambiguous SPIFFE URI SAN.
	ErrInvalidSPIFFEID = errors.New("proxy: certificate has an invalid SPIFFE ID URI SAN")
	// ErrNoPermittedRoute indicates that no valid identity matches policy.
	ErrNoPermittedRoute = errors.New("proxy: no permitted route for SPIFFE identity")
	// ErrNoPeerCertificate indicates that a TLS connection has no peer leaf.
	ErrNoPeerCertificate = errors.New("proxy: TLS connection has no peer certificate")
)

// MTLSRouteLookup resolves a validated identity against the current snapshot.
type MTLSRouteLookup interface {
	Lookup(spiffe.ID) (spiffe.Route, bool)
}

// ResolvedMTLSIdentity associates a validated peer identity with its route.
type ResolvedMTLSIdentity struct {
	Identity spiffe.ID
	Route    spiffe.Route
}

// MTLSIdentityRouter validates workload identities and selects their routes.
// It deliberately does not perform certificate chain verification; callers
// must invoke it only for an already chain-verified peer certificate.
type MTLSIdentityRouter struct {
	routes MTLSRouteLookup
}

// NewMTLSIdentityRouter creates a workload-identity route resolver.
func NewMTLSIdentityRouter(routes MTLSRouteLookup) (*MTLSIdentityRouter, error) {
	if routes == nil {
		return nil, ErrNilRouteLookup
	}
	return &MTLSIdentityRouter{routes: routes}, nil
}

// ResolveCertificate validates all SPIFFE URI SANs and selects the most
// specific permitted route. A malformed SPIFFE SAN rejects the whole
// certificate, even when another SAN would have matched.
func (r *MTLSIdentityRouter) ResolveCertificate(cert *x509.Certificate) (ResolvedMTLSIdentity, error) {
	if r == nil || r.routes == nil {
		return ResolvedMTLSIdentity{}, ErrNilRouteLookup
	}
	if cert == nil {
		return ResolvedMTLSIdentity{}, ErrNoPeerCertificate
	}

	foundSPIFFE := false
	foundRoute := false
	best := ResolvedMTLSIdentity{}
	for _, uri := range cert.URIs {
		if !isSPIFFEURI(uri) {
			continue
		}
		foundSPIFFE = true
		identity, err := spiffe.ParseURL(uri)
		if err != nil {
			return ResolvedMTLSIdentity{}, fmt.Errorf("%w: %w", ErrInvalidSPIFFEID, err)
		}
		route, ok := r.routes.Lookup(identity)
		if !ok {
			continue
		}
		candidate := ResolvedMTLSIdentity{Identity: identity, Route: route}
		if !foundRoute || isBetterIdentity(candidate, best) {
			best = candidate
			foundRoute = true
		}
	}

	if !foundSPIFFE {
		return ResolvedMTLSIdentity{}, ErrNoSPIFFEID
	}
	if !foundRoute {
		return ResolvedMTLSIdentity{}, ErrNoPermittedRoute
	}
	return best, nil
}

// RouteCertificate returns the selected route for a chain-verified peer.
func (r *MTLSIdentityRouter) RouteCertificate(cert *x509.Certificate) (spiffe.Route, error) {
	resolved, err := r.ResolveCertificate(cert)
	if err != nil {
		return spiffe.Route{}, err
	}
	return resolved.Route, nil
}

// ResolveConnection resolves the leaf identity of an established TLS
// connection. It handles certificate rollover naturally because no identity or
// route is cached across connections.
func (r *MTLSIdentityRouter) ResolveConnection(state tls.ConnectionState) (ResolvedMTLSIdentity, error) {
	if len(state.PeerCertificates) == 0 {
		return ResolvedMTLSIdentity{}, ErrNoPeerCertificate
	}
	return r.ResolveCertificate(state.PeerCertificates[0])
}

// VerifyConnection can be installed directly as tls.Config.VerifyConnection.
// The TLS stack must still perform normal certificate-chain verification.
func (r *MTLSIdentityRouter) VerifyConnection(state tls.ConnectionState) error {
	_, err := r.ResolveConnection(state)
	return err
}

func isSPIFFEURI(uri *url.URL) bool {
	return uri != nil && strings.EqualFold(uri.Scheme, "spiffe")
}

func isBetterIdentity(candidate, current ResolvedMTLSIdentity) bool {
	if len(candidate.Route.Path) != len(current.Route.Path) {
		return len(candidate.Route.Path) > len(current.Route.Path)
	}
	if len(candidate.Identity.Path) != len(current.Identity.Path) {
		return len(candidate.Identity.Path) > len(current.Identity.Path)
	}
	if candidate.Identity.TrustDomain != current.Identity.TrustDomain {
		return candidate.Identity.TrustDomain < current.Identity.TrustDomain
	}
	if candidate.Identity.Path != current.Identity.Path {
		return candidate.Identity.Path < current.Identity.Path
	}
	return candidate.Route.Target < current.Route.Target
}
