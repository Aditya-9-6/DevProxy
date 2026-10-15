package waf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sync"
	"testing"
)

func TestCookieEngine_Validate(t *testing.T) {
	secret := "super-secret-key"
	engine := NewCookieEngine(secret)

	payload := []byte("user_id=123")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	cookie := []byte(base64.RawURLEncoding.EncodeToString(payload) + "." + sig)

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{"Valid", cookie, false},
		{"InvalidFormat", []byte("badformat"), true},
		{"InvalidSignature", []byte(string(cookie) + "a"), true},
		{"Empty", []byte(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := engine.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && string(val) != string(payload) {
				t.Errorf("Expected %s, got %s", payload, val)
			}
		})
	}
}

func TestCookieEngine_Concurrency(t *testing.T) {
	engine := NewCookieEngine("secret")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = engine.Validate([]byte("invalid.sig"))
		}()
	}
	wg.Wait()
	if engine.GetFailureCount() != 100 {
		t.Errorf("Expected 100 failures, got %d", engine.GetFailureCount())
	}
}
