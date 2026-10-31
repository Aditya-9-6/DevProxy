package spiffe

import (
	"testing"
)

func FuzzParseID(f *testing.F) {
	f.Add("spiffe://example.org/service")
	f.Add("not-a-spiffe-id")
	f.Fuzz(func(t *testing.T, orig string) {
		_, _ = ParseID(orig)
	})
}
