package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom unexported type for context keys to prevent package-level collisions.
type contextKey int

const (
	UserClaimsKey contextKey = iota
)

// JWKSProvider handles fetching and caching public keys with TTL support.
type JWKSProvider struct {
	mu         sync.RWMutex
	keys       map[string]interface{}
	cache      time.Time
	jwksURL    string
	ttl        time.Duration
	httpClient *http.Client
}

// NewJWKSProvider creates a long-lived JWKSProvider singleton.
func NewJWKSProvider(jwksURL string, ttl time.Duration) *JWKSProvider {
	return &JWKSProvider{
		keys:       make(map[string]interface{}),
		jwksURL:    jwksURL,
		ttl:        ttl,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// FetchKeys simulates or performs fetching of keys from the JWKS endpoint.
func (p *JWKSProvider) FetchKeys(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If a URL is provided and cache is fresh, skip
	if p.jwksURL != "" && time.Since(p.cache) < p.ttl && len(p.keys) > 0 {
		return nil
	}

	// If no URL is set, return without error (useful for testing or manual key injection)
	if p.jwksURL == "" {
		p.cache = time.Now()
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create jwks request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected jwks status: %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kty string   `json:"kty"`
			Kid string   `json:"kid"`
			Use string   `json:"use"`
			N   string   `json:"n"`
			E   string   `json:"e"`
			X5c []string `json:"x5c"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode jwks: %w", err)
	}

	newKeys := make(map[string]interface{})
	for _, k := range jwks.Keys {
		newKeys[k.Kid] = k
	}

	p.keys = newKeys
	p.cache = time.Now()
	return nil
}

// SetKey allows manual injection of keys for testing or static configuration.
func (p *JWKSProvider) SetKey(kid string, key interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keys[kid] = key
}

// ValidateToken performs cryptographic signature verification with proper error differentiation.
func (p *JWKSProvider) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	_ = p.FetchKeys(context.Background())

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			if _, okECDSA := token.Method.(*jwt.SigningMethodECDSA); !okECDSA {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
		}

		p.mu.RLock()
		defer p.mu.RUnlock()

		keyID, _ := token.Header["kid"].(string)
		if key, ok := p.keys[keyID]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("key not found for kid: %s", keyID)
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("token is expired: %w", err)
		}
		return nil, fmt.Errorf("malformed or invalid token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token signature")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims structure")
	}

	return claims, nil
}

// ClaimsExtractor extracts claims into context for downstream WAF rules using a custom unexported type.
func ClaimsExtractor(ctx context.Context, claims jwt.MapClaims) context.Context {
	return context.WithValue(ctx, UserClaimsKey, claims)
}
