package analysis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSProvider handles fetching and caching public keys.
type JWKSProvider struct {
	mu    sync.RWMutex
	keys  map[string]interface{}
	cache time.Time
}

var jwksPool = sync.Pool{
	New: func() interface{} {
		return &JWKSProvider{keys: make(map[string]interface{})}
	},
}

// ValidateToken performs cryptographic signature verification.
func (p *JWKSProvider) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		p.mu.RLock()
		defer p.mu.RUnlock()

		keyID, _ := token.Header["kid"].(string)
		if key, ok := p.keys[keyID]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("key not found: %s", keyID)
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return token.Claims.(jwt.MapClaims), nil
}

// ClaimsExtractor extracts claims into context for downstream WAF rules.
func ClaimsExtractor(ctx context.Context, claims jwt.MapClaims) context.Context {
	return context.WithValue(ctx, "user_claims", claims)
}
