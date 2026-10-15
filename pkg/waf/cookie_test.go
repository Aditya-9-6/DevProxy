package waf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestCookieEngine_Validate(t *testing.T) {
	secret := "super-secret-key"
	engine := NewCookieEngine(secret)

	payload := "user_id=123"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	cookie := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig

	// Test Success
	val, err := engine.Validate(cookie)
	if err != nil {
		t.Fatalf("Expected success, got %v", err)
	}
	if val != payload {
		t.Errorf("Expected %s, got %s", payload, val)
	}

	// Test Invalid Signature
	_, err = engine.Validate(cookie + "invalid")
	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}

	// Test Failure Counter
	if engine.GetFailureCount() == 0 {
		t.Error("Expected failure count > 0")
	}
}
