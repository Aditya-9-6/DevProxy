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

type contextKey int

const (
	UserClaimsKey contextKey = iota
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

type JWKSProvider struct {
	mu         sync.RWMutex
	keys       map[string]interface{}
	cache      time.Time
	jwksURL    string
	ttl        time.Duration
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewJWKSProvider(jwksURL string, ttl time.Duration) *JWKSProvider {
	ctx, cancel := context.WithCancel(context.Background())
	p := &JWKSProvider{
		keys:       make(map[string]interface{}),
		jwksURL:    jwksURL,
		ttl:        ttl,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		ctx:        ctx,
		cancel:     cancel,
	}
	if jwksURL != "" {
		go p.backgroundRefresh()
	}
	return p
}

func (p *JWKSProvider) backgroundRefresh() {
	ticker := time.NewTicker(p.ttl / 2)
	defer ticker.Stop()
	_ = p.FetchKeys(p.ctx)
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			_ = p.FetchKeys(p.ctx)
		}
	}
}

func (p *JWKSProvider) FetchKeys(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	newKeys := make(map[string]interface{})
	for _, k := range jwks.Keys {
		newKeys[k.Kid] = k
	}

	p.keys = newKeys
	p.cache = time.Now()
	return nil
}

func (p *JWKSProvider) SetKey(kid string, key interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keys[kid] = key
}

func (p *JWKSProvider) ValidateToken(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		p.mu.RLock()
		defer p.mu.RUnlock()
		keyID, _ := token.Header["kid"].(string)
		if key, ok := p.keys[keyID]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("key not found")
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func ClaimsExtractor(ctx context.Context, claims jwt.MapClaims) context.Context {
	return context.WithValue(ctx, UserClaimsKey, claims)
}
