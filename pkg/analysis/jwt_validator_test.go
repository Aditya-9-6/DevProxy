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
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	provider := NewJWKSProvider("", time.Hour)
	provider.SetKey("test-kid", &privateKey.PublicKey)

	// Valid Token
	claims := jwt.MapClaims{"sub": "123", "exp": time.Now().Add(time.Hour).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	tokenString, _ := token.SignedString(privateKey)

	_, err := provider.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Errorf("expected valid token, got: %v", err)
	}

	// Expired Token
	expiredClaims := jwt.MapClaims{"sub": "123", "exp": time.Now().Add(-time.Hour).Unix()}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS256, expiredClaims)
	expiredToken.Header["kid"] = "test-kid"
	expiredString, _ := expiredToken.SignedString(privateKey)

	_, err = provider.ValidateToken(context.Background(), expiredString)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestJWKSProvider_AlgorithmConfusion(t *testing.T) {
	t.Parallel()
	provider := NewJWKSProvider("", time.Hour)
	// Attempt to use HS256 (symmetric) when expecting RS256 (asymmetric)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "123"})
	tokenString, _ := token.SignedString([]byte("secret"))

	_, err := provider.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("expected error for invalid signing method, got nil")
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
			provider.SetKey("kid", id)
			_, _ = provider.ValidateToken(context.Background(), "invalid")
		}(i)
	}
	wg.Wait()
}
