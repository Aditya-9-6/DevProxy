package spiffe

import (
	"errors"
	"net/url"
	"sync"
	"testing"
)

func TestParseURLConcurrent(t *testing.T) {
	t.Parallel()

	uri, err := url.Parse("spiffe://example.org/workload")
	if err != nil {
		t.Fatalf("url.Parse() unexpected error: %v", err)
	}

	type parseResult struct {
		id  ID
		err error
	}

	const workers = 64
	results := make(chan parseResult, workers)
	var waitGroup sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			id, parseErr := ParseURL(uri)
			results <- parseResult{id: id, err: parseErr}
		}()
	}

	waitGroup.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			t.Errorf("ParseURL() unexpected error: %v", result.err)
			continue
		}
		if result.id.TrustDomain != "example.org" || result.id.Path != "/workload" {
			t.Errorf("ParseURL() = %+v, want example.org/workload", result.id)
		}
	}
}

func TestValidatePathCanonicalEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{name: "simple", path: "/api/v1"},
		{name: "reserved character", path: "/service:v1"},
		{name: "decoded space", path: "/hello world"},
		{name: "encoded space", path: "/hello%20world"},
		{name: "encoded reserved", path: "/service%3Av1"},
		{name: "encoded utf8", path: "/caf%C3%A9"},
		{name: "decoded utf8", path: "/café"},
		{name: "empty", path: "", wantErr: ErrInvalidPath},
		{name: "relative", path: "api/v1", wantErr: ErrInvalidPath},
		{name: "dot segment", path: "/a/../b", wantErr: ErrDotSegment},
		{name: "encoded separator", path: "/a%2Fb", wantErr: ErrEncodedPathSeparator},
		{name: "lowercase separator", path: "/a%2fb", wantErr: ErrNonCanonicalPercentEncoding},
		{name: "encoded unreserved", path: "/%41", wantErr: ErrNonCanonicalPercentEncoding},
		{name: "encoded backslash", path: "/a%5Cb", wantErr: ErrInvalidPath},
		{name: "malformed escape", path: "/%zz", wantErr: ErrInvalidPath},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePath(test.path)
			if test.wantErr == nil && err != nil {
				t.Fatalf("ValidatePath(%q) unexpected error: %v", test.path, err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidatePath(%q) error = %v, want %v", test.path, err, test.wantErr)
			}
		})
	}
}
