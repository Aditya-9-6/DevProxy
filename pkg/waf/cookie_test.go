package waf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sync"
	"testing"
	"unsafe"
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
			dst := make([]byte, 1024)
			n, err := engine.Validate(tt.input, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && string(dst[:n]) != string(payload) {
				t.Errorf("Expected %s, got %s", payload, dst[:n])
			}
		})
	}
}

func TestCacheLinePad(t *testing.T) {
	var p CacheLinePad
	if unsafe.Sizeof(p) < 64 {
		t.Errorf("CacheLinePad size too small: %d", unsafe.Sizeof(p))
	}
}

func TestCookieEngine_Concurrency(t *testing.T) {
	engine := NewCookieEngine("secret")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = engine.Validate([]byte("invalid.sig"), make([]byte, 1024))
		}()
	}
	wg.Wait()
	if engine.GetFailureCount() != 100 {
		t.Errorf("Expected 100 failures, got %d", engine.GetFailureCount())
	}
}

func BenchmarkCookieValidate(b *testing.B) {
	secret := "super-secret-key"
	engine := NewCookieEngine(secret)
	payload := []byte("user_id=123")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	cookie := []byte(base64.RawURLEncoding.EncodeToString(payload) + "." + sig)
	dst := make([]byte, 1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Validate(cookie, dst)
	}
}
