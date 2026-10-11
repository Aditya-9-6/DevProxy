package analysis

import (
	"testing"
)

func TestJWKSProvider_ValidateToken(t *testing.T) {
	provider := &JWKSProvider{keys: make(map[string]interface{})}

	// Test case: missing key in cache
	_, err := provider.ValidateToken("header.payload.sig")
	if err == nil {
		t.Error("Expected error for missing key, got nil")
	}
}
