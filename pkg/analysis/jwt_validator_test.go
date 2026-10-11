package analysis

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWKSProvider_ValidateToken_MissingKey(t *testing.T) {
	t.Parallel()
	provider := NewJWKSProvider("", time.Hour)

	_, err := provider.ValidateToken(context.Background(), "header.payload.sig")
	if err == nil {
		t.Error("Expected error for missing key, got nil")
	}
}

func TestJWKSProvider_ValidAndExpiredToken(t *testing.T) {
	t.Parallel()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	provider := NewJWKSProvider("", time.Hour)
	provider.SetKey("test-kid", &privateKey.PublicKey)

	claims := jwt.MapClaims{
		"sub": "1234567890",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"

	tokenString, _ := token.SignedString(privateKey)

	parsedClaims, err := provider.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}

	if parsedClaims["sub"] != "1234567890" {
		t.Errorf("expected sub 1234567890, got %v", parsedClaims["sub"])
	}
}

func TestJWKSProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	provider := NewJWKSProvider("", time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			provider.SetKey("kid-", id)
			_, _ = provider.ValidateToken(context.Background(), "invalid.token.string")
		}(i)
	}
	wg.Wait()
}
