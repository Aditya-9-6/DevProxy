package security

import "testing"

func TestCookieFilter(t *testing.T) {
	filter := NewCookieFilter()
	sig := "signed-cookie-123"

	if filter.Contains(sig) {
		t.Errorf("Filter should not contain signature before insertion")
	}

	filter.Add(sig)

	if !filter.Contains(sig) {
		t.Errorf("Filter should contain signature after insertion")
	}

	if filter.Contains("unknown-sig") {
		t.Errorf("Filter should not contain unknown signature")
	}
}
