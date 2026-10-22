package spiffe

import (
	"errors"
	"net/url"
	"testing"
)

var allocationID ID

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantDomain string
		wantPath   string
		wantErr    error
	}{
		{name: "workload", raw: "spiffe://example.org/workload", wantDomain: "example.org", wantPath: "/workload"},
		{name: "nested path", raw: "spiffe://example.org/api/v1", wantDomain: "example.org", wantPath: "/api/v1"},
		{name: "root without slash", raw: "spiffe://example.org", wantDomain: "example.org", wantPath: "/"},
		{name: "root with slash", raw: "spiffe://example.org/", wantDomain: "example.org", wantPath: "/"},
		{name: "encoded reserved", raw: "spiffe://example.org/service%3Av1", wantDomain: "example.org", wantPath: "/service:v1"},
		{name: "encoded space", raw: "spiffe://example.org/hello%20world", wantDomain: "example.org", wantPath: "/hello world"},
		{name: "encoded utf8", raw: "spiffe://example.org/caf%C3%A9", wantDomain: "example.org", wantPath: "/café"},
		{name: "http scheme", raw: "https://example.org/workload", wantErr: ErrInvalidScheme},
		{name: "uppercase scheme", raw: "SPIFFE://example.org/workload", wantErr: ErrInvalidScheme},
		{name: "opaque URI", raw: "spiffe:example.org/workload", wantErr: ErrInvalidURIForm},
		{name: "userinfo", raw: "spiffe://user@example.org/workload", wantErr: ErrInvalidUserInfo},
		{name: "port", raw: "spiffe://example.org:443/workload", wantErr: ErrInvalidPort},
		{name: "empty port", raw: "spiffe://example.org:/workload", wantErr: ErrInvalidPort},
		{name: "query", raw: "spiffe://example.org/workload?role=admin", wantErr: ErrInvalidQuery},
		{name: "empty query", raw: "spiffe://example.org/workload?", wantErr: ErrInvalidQuery},
		{name: "fragment", raw: "spiffe://example.org/workload#fragment", wantErr: ErrInvalidFragment},
		{name: "empty fragment", raw: "spiffe://example.org/workload#", wantErr: ErrInvalidFragment},
		{name: "dot segment", raw: "spiffe://example.org/a/../b", wantErr: ErrDotSegment},
		{name: "encoded dot segment", raw: "spiffe://example.org/%2e%2e/b", wantErr: ErrDotSegment},
		{name: "mixed encoded dot segment", raw: "spiffe://example.org/.%2E/b", wantErr: ErrDotSegment},
		{name: "encoded slash", raw: "spiffe://example.org/a%2Fb", wantErr: ErrEncodedPathSeparator},
		{name: "encoded backslash", raw: "spiffe://example.org/a%5Cb", wantErr: ErrInvalidPath},
		{name: "lowercase percent", raw: "spiffe://example.org/a%2fb", wantErr: ErrNonCanonicalPercentEncoding},
		{name: "encoded unreserved", raw: "spiffe://example.org/%41", wantErr: ErrNonCanonicalPercentEncoding},
		{name: "malformed percent", raw: "spiffe://example.org/%zz", wantErr: ErrInvalidURI},
		{name: "uppercase trust domain", raw: "spiffe://Example.org/workload", wantErr: ErrInvalidTrustDomain},
		{name: "trailing trust dot", raw: "spiffe://example.org./workload", wantErr: ErrInvalidTrustDomain},
		{name: "invalid label", raw: "spiffe://-example.org/workload", wantErr: ErrInvalidTrustDomain},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(test.raw)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Parse(%q) error = %v, want %v", test.raw, err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", test.raw, err)
			}
			if got.TrustDomain != test.wantDomain || got.Path != test.wantPath {
				t.Fatalf("Parse(%q) = %+v, want %s%s", test.raw, got, test.wantDomain, test.wantPath)
			}
		})
	}
}

func TestParseURLRejectsForgedFields(t *testing.T) {
	tests := []struct {
		name    string
		uri     *url.URL
		wantErr error
	}{
		{name: "nil", wantErr: ErrInvalidURI},
		{name: "force query", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/x", ForceQuery: true}, wantErr: ErrInvalidQuery},
		{name: "fragment", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/x", Fragment: "x"}, wantErr: ErrInvalidFragment},
		{name: "user", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/x", User: url.User("alice")}, wantErr: ErrInvalidUserInfo},
		{name: "opaque", uri: &url.URL{Scheme: "spiffe", Opaque: "example.org/x"}, wantErr: ErrInvalidURIForm},
		{name: "port", uri: &url.URL{Scheme: "spiffe", Host: "example.org:443", Path: "/x"}, wantErr: ErrInvalidPort},
		{name: "inconsistent raw path", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/a", RawPath: "/b"}, wantErr: ErrInvalidPath},
		{name: "encoded separator", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/a/b", RawPath: "/a%2Fb"}, wantErr: ErrEncodedPathSeparator},
		{name: "noncanonical escape", uri: &url.URL{Scheme: "spiffe", Host: "example.org", Path: "/A", RawPath: "/%41"}, wantErr: ErrNonCanonicalPercentEncoding},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseURL(test.uri)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ParseURL() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestValidators(t *testing.T) {
	validDomains := []string{"example.org", "localhost", "a", "service-1.example.org"}
	for _, domain := range validDomains {
		if err := ValidateTrustDomain(domain); err != nil {
			t.Errorf("ValidateTrustDomain(%q) unexpected error: %v", domain, err)
		}
	}
	invalidDomains := []string{"", "Example.org", "example.org.", ".example.org", "bad_label", "a..b"}
	for _, domain := range invalidDomains {
		if err := ValidateTrustDomain(domain); !errors.Is(err, ErrInvalidTrustDomain) {
			t.Errorf("ValidateTrustDomain(%q) error = %v", domain, err)
		}
	}

	if err := ValidatePath("/api/v1"); err != nil {
		t.Errorf("ValidatePath() unexpected error: %v", err)
	}
	invalidPaths := []string{"", "api", "/a/../b", "/a" + string(rune(backslashByte)) + "b"}
	for _, path := range invalidPaths {
		if err := ValidatePath(path); err == nil {
			t.Errorf("ValidatePath(%q) unexpectedly succeeded", path)
		}
	}
}

func TestIDString(t *testing.T) {
	id := ID{TrustDomain: "example.org", Path: "/workload"}
	if got, want := id.String(), "spiffe://example.org/workload"; got != want {
		t.Fatalf("ID.String() = %q, want %q", got, want)
	}
}

func TestParseURLAllocations(t *testing.T) {
	uri, err := url.Parse("spiffe://example.org/workload")
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		var parseErr error
		allocationID, parseErr = ParseURL(uri)
		if parseErr != nil {
			panic(parseErr)
		}
	})
	if allocs != 0 {
		t.Fatalf("ParseURL allocated %v objects, want 0", allocs)
	}
}

func BenchmarkParseURL(b *testing.B) {
	uri, err := url.Parse("spiffe://example.org/workload")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id, parseErr := ParseURL(uri)
		if parseErr != nil {
			b.Fatal(parseErr)
		}
		allocationID = id
	}
}
