package spiffe

import (
	"net/url"
	"testing"
)

func FuzzSPIFFESAN(f *testing.F) {
	seeds := []string{
		"spiffe://example.org/workload",
		"spiffe://example.org/api/v1",
		"spiffe://user@example.org/workload",
		"spiffe://example.org:443/workload",
		"spiffe://example.org/workload?query=1",
		"spiffe://example.org/workload#fragment",
		"spiffe://example.org/a/../b",
		"spiffe://example.org/%2e%2e/b",
		"spiffe://example.org/a%2fb",
		"spiffe://example.org/%41",
		"spiffe://example.org/%zz",
		"SPIFFE://example.org/workload",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := url.Parse(raw)
		if err == nil {
			if identity, parseErr := ParseURL(parsed); parseErr == nil {
				assertFuzzIdentity(t, identity)
			}
		}

		identity, err := Parse(raw)
		if err == nil {
			assertFuzzIdentity(t, identity)
		}
	})
}

func assertFuzzIdentity(t *testing.T, identity ID) {
	t.Helper()
	if err := ValidateTrustDomain(identity.TrustDomain); err != nil {
		t.Fatalf("valid identity has invalid trust domain: %v", err)
	}
	if err := ValidatePath(identity.Path); err != nil {
		t.Fatalf("valid identity has invalid path: %v", err)
	}

	reparsed, err := url.Parse(identity.String())
	if err != nil {
		t.Fatalf("canonical identity cannot be reparsed: %v", err)
	}
	copy, err := ParseURL(reparsed)
	if err != nil {
		t.Fatalf("canonical identity cannot be revalidated: %v", err)
	}
	if copy != identity {
		t.Fatalf("canonical identity changed: got %+v, want %+v", copy, identity)
	}
}
