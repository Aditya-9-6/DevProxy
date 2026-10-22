package spiffe

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

var policyAllocationID ID

func TestNewPolicy(t *testing.T) {
	policy, err := NewPolicy(
		[]string{"example.org"},
		[]Route{{TrustDomain: "example.org", Path: "/api/", Target: "backend"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Allows(ID{TrustDomain: "example.org"}) {
		t.Fatal("expected exact trust domain to be allowed")
	}
	if policy.Allows(ID{TrustDomain: "example.org.evil"}) {
		t.Fatal("suffix-like trust domain was allowed")
	}
	routes := policy.Routes()
	if len(routes) != 1 || routes[0].Path != "/api" {
		t.Fatalf("unexpected routes: %+v", routes)
	}
}

func TestPolicyRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		domains []string
		routes  []Route
		wantErr error
	}{
		{name: "duplicate domain", domains: []string{"example.org", "example.org"}, wantErr: ErrDuplicateTrustDomain},
		{name: "invalid domain", domains: []string{"Example.org"}, wantErr: ErrInvalidTrustDomain},
		{name: "route domain denied", domains: []string{"example.org"}, routes: []Route{{TrustDomain: "other.org", Path: "/api"}}, wantErr: ErrRouteTrustDomainNotAllowed},
		{name: "invalid route domain", domains: []string{"example.org"}, routes: []Route{{TrustDomain: "bad_domain", Path: "/api"}}, wantErr: ErrInvalidRoute},
		{name: "invalid route path", domains: []string{"example.org"}, routes: []Route{{TrustDomain: "example.org", Path: "api"}}, wantErr: ErrInvalidRoute},
		{name: "duplicate route", domains: []string{"example.org"}, routes: []Route{{TrustDomain: "example.org", Path: "/api"}, {TrustDomain: "example.org", Path: "/api"}}, wantErr: ErrDuplicateRoute},
		{name: "duplicate normalized route", domains: []string{"example.org"}, routes: []Route{{TrustDomain: "example.org", Path: "/api"}, {TrustDomain: "example.org", Path: "/api/"}}, wantErr: ErrDuplicateRoute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPolicy(test.domains, test.routes)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NewPolicy() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestPolicyCopiesInputs(t *testing.T) {
	domains := []string{"example.org"}
	routes := []Route{{TrustDomain: "example.org", Path: "/api", Target: "backend"}}
	policy, err := NewPolicy(domains, routes)
	if err != nil {
		t.Fatal(err)
	}

	domains[0] = "evil.org"
	routes[0].Path = "/evil"
	copyOfRoutes := policy.Routes()
	copyOfRoutes[0].Target = "changed"

	stored := policy.Routes()
	if stored[0].Path != "/api" || stored[0].Target != "backend" {
		t.Fatalf("policy was mutated through caller state: %+v", stored)
	}
}

func TestPolicyConcurrentReads(t *testing.T) {
	policy, err := NewPolicy([]string{"example.org"}, []Route{{TrustDomain: "example.org", Path: "/api"}})
	if err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	var failed atomic.Bool
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := 0; i < 1000; i++ {
				if !policy.Allows(ID{TrustDomain: "example.org"}) {
					failed.Store(true)
					return
				}
			}
		}()
	}
	wait.Wait()
	if failed.Load() {
		t.Fatal("concurrent policy read observed an invalid decision")
	}
}

func TestPolicyAllowsAllocations(t *testing.T) {
	policy, err := NewPolicy([]string{"example.org"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		policyAllocationID = ID{TrustDomain: "example.org"}
		if !policy.Allows(policyAllocationID) {
			panic("policy denied exact domain")
		}
	})
	if allocs != 0 {
		t.Fatalf("Policy.Allows allocated %v objects, want 0", allocs)
	}
}
