package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

type pr445RouteLookup struct {
	routes map[string]spiffe.Route
}

func (l pr445RouteLookup) Lookup(identity spiffe.ID) (spiffe.Route, bool) {
	route, ok := l.routes[identity.String()]
	return route, ok
}

func pr445ParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return parsed
}

func pr445Certificate(t *testing.T, rawURIs ...string) *x509.Certificate {
	t.Helper()
	uris := make([]*url.URL, 0, len(rawURIs))
	for _, raw := range rawURIs {
		uris = append(uris, pr445ParseURL(t, raw))
	}
	return &x509.Certificate{URIs: uris}
}

func TestPR445MTLSIdentityRouterConstructor(t *testing.T) {
	t.Parallel()

	router, err := NewMTLSIdentityRouter(nil)
	if router != nil {
		t.Fatalf("NewMTLSIdentityRouter(nil) router = %v, want nil", router)
	}
	if !errors.Is(err, ErrNilRouteLookup) {
		t.Fatalf("NewMTLSIdentityRouter(nil) error = %v, want %v", err, ErrNilRouteLookup)
	}
}

func TestPR445MTLSIdentityRouterErrorBranches(t *testing.T) {
	t.Parallel()

	permittedLookup := pr445RouteLookup{
		routes: map[string]spiffe.Route{
			"spiffe://example.org/workload": spiffe.Route{Path: "/", Target: "permitted"},
		},
	}
	tests := []struct {
		name           string
		nilCertificate bool
		uris           []string
		lookup         pr445RouteLookup
		want           error
	}{
		{name: "nil certificate", nilCertificate: true, want: ErrNoPeerCertificate},
		{name: "certificate without URI SANs", want: ErrNoSPIFFEID},
		{name: "non-SPIFFE URI only", uris: []string{"https://example.org/workload"}, want: ErrNoSPIFFEID},
		{name: "malformed SPIFFE URI", uris: []string{"spiffe://example.org/workload?role=admin"}, want: ErrInvalidSPIFFEID},
		{name: "no permitted route", uris: []string{"spiffe://example.org/workload"}, want: ErrNoPermittedRoute},
		{
			name:   "malformed SAN rejects otherwise matching certificate",
			uris:   []string{"spiffe://example.org/workload", "spiffe://example.org/workload?role=admin"},
			lookup: permittedLookup,
			want:   ErrInvalidSPIFFEID,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			router, err := NewMTLSIdentityRouter(test.lookup)
			if err != nil {
				t.Fatalf("NewMTLSIdentityRouter() error = %v", err)
			}

			var certificate *x509.Certificate
			if !test.nilCertificate {
				certificate = pr445Certificate(t, test.uris...)
			}
			_, err = router.ResolveCertificate(certificate)
			if !errors.Is(err, test.want) {
				t.Fatalf("ResolveCertificate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestPR445MTLSIdentityRouterNilState(t *testing.T) {
	t.Parallel()

	var nilRouter *MTLSIdentityRouter
	_, err := nilRouter.ResolveCertificate(nil)
	if !errors.Is(err, ErrNilRouteLookup) {
		t.Fatalf("nil router error = %v, want %v", err, ErrNilRouteLookup)
	}

	emptyRouter := &MTLSIdentityRouter{}
	_, err = emptyRouter.ResolveCertificate(nil)
	if !errors.Is(err, ErrNilRouteLookup) {
		t.Fatalf("empty router error = %v, want %v", err, ErrNilRouteLookup)
	}
}

func TestPR445MTLSIdentityRouterSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		routes       map[string]spiffe.Route
		uris         []string
		wantIdentity string
		wantTarget   string
	}{
		{
			name: "route path takes precedence",
			routes: map[string]spiffe.Route{
				"spiffe://example.org/workload":        spiffe.Route{Path: "/api/v1", Target: "workload"},
				"spiffe://example.org/workload/nested": spiffe.Route{Path: "/api", Target: "nested"},
			},
			uris:         []string{"spiffe://example.org/workload", "spiffe://example.org/workload/nested"},
			wantIdentity: "spiffe://example.org/workload",
			wantTarget:   "workload",
		},
		{
			name: "identity path breaks route path tie",
			routes: map[string]spiffe.Route{
				"spiffe://example.org/workload":        spiffe.Route{Path: "/api", Target: "workload"},
				"spiffe://example.org/workload/nested": spiffe.Route{Path: "/api", Target: "nested"},
			},
			uris:         []string{"spiffe://example.org/workload", "spiffe://example.org/workload/nested"},
			wantIdentity: "spiffe://example.org/workload/nested",
			wantTarget:   "nested",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			router, err := NewMTLSIdentityRouter(pr445RouteLookup{routes: test.routes})
			if err != nil {
				t.Fatalf("NewMTLSIdentityRouter() error = %v", err)
			}

			resolved, err := router.ResolveCertificate(pr445Certificate(t, test.uris...))
			if err != nil {
				t.Fatalf("ResolveCertificate() error = %v", err)
			}
			if got := resolved.Identity.String(); got != test.wantIdentity {
				t.Fatalf("resolved identity = %q, want %q", got, test.wantIdentity)
			}
			if got := resolved.Route.Target; got != test.wantTarget {
				t.Fatalf("resolved target = %q, want %q", got, test.wantTarget)
			}
		})
	}
}

func TestPR445IsBetterIdentity(t *testing.T) {
	t.Parallel()

	sameID := spiffe.ID{TrustDomain: "example.org", Path: "/workload"}
	sameRoute := spiffe.Route{Path: "/api", Target: "target"}
	tests := []struct {
		name      string
		candidate ResolvedMTLSIdentity
		current   ResolvedMTLSIdentity
		want      bool
	}{
		{
			name:      "longer route path wins",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api/v1", Target: "target"}},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			want:      true,
		},
		{
			name:      "shorter route path loses",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api/v1", Target: "target"}},
			want:      false,
		},
		{
			name:      "longer identity path wins",
			candidate: ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/workload/nested"}, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			want:      true,
		},
		{
			name:      "shorter identity path loses",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/workload/nested"}, Route: sameRoute},
			want:      false,
		},
		{
			name:      "lower trust domain wins",
			candidate: ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "a.example.org", Path: "/workload"}, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "b.example.org", Path: "/workload"}, Route: sameRoute},
			want:      true,
		},
		{
			name:      "higher trust domain loses",
			candidate: ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "b.example.org", Path: "/workload"}, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "a.example.org", Path: "/workload"}, Route: sameRoute},
			want:      false,
		},
		{
			name:      "lower identity path wins",
			candidate: ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/alpha"}, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/beta"}, Route: sameRoute},
			want:      true,
		},
		{
			name:      "higher identity path loses",
			candidate: ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/beta"}, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: spiffe.ID{TrustDomain: "example.org", Path: "/alpha"}, Route: sameRoute},
			want:      false,
		},
		{
			name:      "lower route target wins",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api", Target: "a"}},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api", Target: "b"}},
			want:      true,
		},
		{
			name:      "higher route target loses",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api", Target: "b"}},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: spiffe.Route{Path: "/api", Target: "a"}},
			want:      false,
		},
		{
			name:      "equal identities do not win",
			candidate: ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			current:   ResolvedMTLSIdentity{Identity: sameID, Route: sameRoute},
			want:      false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := isBetterIdentity(test.candidate, test.current); got != test.want {
				t.Fatalf("isBetterIdentity() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPR445MTLSIdentityRouterConnectionAPIs(t *testing.T) {
	t.Parallel()

	lookup := pr445RouteLookup{
		routes: map[string]spiffe.Route{
			"spiffe://example.org/workload": spiffe.Route{Path: "/", Target: "permitted"},
		},
	}
	router, err := NewMTLSIdentityRouter(lookup)
	if err != nil {
		t.Fatalf("NewMTLSIdentityRouter() error = %v", err)
	}
	certificate := pr445Certificate(t, "spiffe://example.org/workload")
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{certificate}}

	resolved, err := router.ResolveConnection(state)
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}
	if resolved.Route.Target != "permitted" {
		t.Fatalf("ResolveConnection() target = %q, want permitted", resolved.Route.Target)
	}

	route, err := router.RouteCertificate(certificate)
	if err != nil {
		t.Fatalf("RouteCertificate() error = %v", err)
	}
	if route.Target != "permitted" {
		t.Fatalf("RouteCertificate() target = %q, want permitted", route.Target)
	}
	if err := router.VerifyConnection(state); err != nil {
		t.Fatalf("VerifyConnection() error = %v", err)
	}

	_, err = router.ResolveConnection(tls.ConnectionState{})
	if !errors.Is(err, ErrNoPeerCertificate) {
		t.Fatalf("ResolveConnection() empty state error = %v, want %v", err, ErrNoPeerCertificate)
	}
}

func TestPR445IsSPIFFEURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		uri  *url.URL
		want bool
	}{
		{name: "nil URI", uri: nil, want: false},
		{name: "HTTPS URI", uri: &url.URL{Scheme: "https"}, want: false},
		{name: "SPIFFE URI", uri: &url.URL{Scheme: "spiffe"}, want: true},
		{name: "uppercase SPIFFE URI", uri: &url.URL{Scheme: "SPIFFE"}, want: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := isSPIFFEURI(test.uri); got != test.want {
				t.Fatalf("isSPIFFEURI() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPR445MTLSIdentityRouterConcurrentResolution(t *testing.T) {
	t.Parallel()

	lookup := pr445RouteLookup{
		routes: map[string]spiffe.Route{
			"spiffe://example.org/workload": spiffe.Route{Path: "/", Target: "permitted"},
		},
	}
	router, err := NewMTLSIdentityRouter(lookup)
	if err != nil {
		t.Fatalf("NewMTLSIdentityRouter() error = %v", err)
	}
	certificate := pr445Certificate(t, "spiffe://example.org/workload")

	const workerCount = 16
	const iterationCount = 100
	errorsCh := make(chan string, workerCount)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for iteration := 0; iteration < iterationCount; iteration++ {
				resolved, resolveErr := router.ResolveCertificate(certificate)
				if resolveErr != nil {
					errorsCh <- resolveErr.Error()
					return
				}
				if resolved.Route.Target != "permitted" {
					errorsCh <- "resolved route target is not permitted"
					return
				}
			}
		}()
	}
	waitGroup.Wait()
	close(errorsCh)

	for message := range errorsCh {
		t.Errorf("concurrent resolution failed: %s", message)
	}
}
