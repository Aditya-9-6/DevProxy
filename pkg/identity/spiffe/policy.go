package spiffe

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrDuplicateTrustDomain indicates a repeated trust-domain authorization.
	ErrDuplicateTrustDomain = errors.New("spiffe: duplicate trust domain")
	// ErrRouteTrustDomainNotAllowed indicates a route outside the trust-domain allowlist.
	ErrRouteTrustDomainNotAllowed = errors.New("spiffe: route trust domain is not allowed")
	// ErrDuplicateRoute indicates repeated routing policy for one trust domain and path.
	ErrDuplicateRoute = errors.New("spiffe: duplicate route")
	// ErrInvalidRoute indicates malformed route policy.
	ErrInvalidRoute = errors.New("spiffe: invalid route")
)

// Route describes one trust-domain-scoped workload path and its destination.
// Path is a decoded path prefix; trailing slashes are normalized during
// policy construction.
type Route struct {
	TrustDomain string
	Path        string
	Target      string
}

// Policy is an immutable SPIFFE authorization and routing policy. All
// constructor inputs are copied, and no exported method exposes mutable state.
type Policy struct {
	trustDomains map[string]struct{}
	routes       []Route
}

// NewPolicy validates and copies a trust-domain allowlist and route set.
func NewPolicy(trustDomains []string, routes []Route) (*Policy, error) {
	domains, err := copyTrustDomains(trustDomains)
	if err != nil {
		return nil, err
	}
	copiedRoutes, err := copyRoutes(domains, routes)
	if err != nil {
		return nil, err
	}
	return &Policy{trustDomains: domains, routes: copiedRoutes}, nil
}

// Allows reports whether the identity belongs to an explicitly authorized
// trust domain. The comparison is exact and never performs suffix matching.
func (p *Policy) Allows(id ID) bool {
	if p == nil {
		return false
	}
	_, ok := p.trustDomains[id.TrustDomain]
	return ok
}

// Routes returns a copy of the immutable route definitions.
func (p *Policy) Routes() []Route {
	if p == nil {
		return nil
	}
	routes := make([]Route, len(p.routes))
	copy(routes, p.routes)
	return routes
}

// TrustDomains returns a copy of the authorized trust domains.
func (p *Policy) TrustDomains() []string {
	if p == nil {
		return nil
	}
	domains := make([]string, 0, len(p.trustDomains))
	for domain := range p.trustDomains {
		domains = append(domains, domain)
	}
	return domains
}

// RouteCount returns the number of configured route prefixes.
func (p *Policy) RouteCount() int {
	if p == nil {
		return 0
	}
	return len(p.routes)
}

func copyTrustDomains(domains []string) (map[string]struct{}, error) {
	copied := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		if err := ValidateTrustDomain(domain); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidTrustDomain, err)
		}
		if _, exists := copied[domain]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateTrustDomain, domain)
		}
		copied[domain] = struct{}{}
	}
	return copied, nil
}

type routeKey struct {
	trustDomain string
	path        string
}

func copyRoutes(allowed map[string]struct{}, routes []Route) ([]Route, error) {
	copied := make([]Route, 0, len(routes))
	seen := make(map[routeKey]struct{}, len(routes))
	for _, route := range routes {
		if err := validateRoute(route, allowed); err != nil {
			return nil, err
		}
		route.Path = normalizeRoutePath(route.Path)
		key := routeKey{trustDomain: route.TrustDomain, path: route.Path}
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: %s%s", ErrDuplicateRoute, route.TrustDomain, route.Path)
		}
		seen[key] = struct{}{}
		copied = append(copied, route)
	}
	return copied, nil
}

func validateRoute(route Route, allowed map[string]struct{}) error {
	if err := ValidateTrustDomain(route.TrustDomain); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRoute, err)
	}
	if _, ok := allowed[route.TrustDomain]; !ok {
		return fmt.Errorf("%w: %s", ErrRouteTrustDomainNotAllowed, route.TrustDomain)
	}
	if err := ValidatePath(route.Path); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRoute, err)
	}
	return nil
}

func normalizeRoutePath(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		return "/"
	}
	return trimmed
}
