package waf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync/atomic"
)

var (
	ErrInvalidCookieFormat = errors.New("invalid cookie format")
	ErrInvalidSignature    = errors.New("invalid signature")
)

type CookieEngine struct {
	secretKey      []byte
	FailureCounter CacheLinePad
}

func NewCookieEngine(secretKey string) *CookieEngine {
	return &CookieEngine{
		secretKey: []byte(secretKey),
	}
}

func (e *CookieEngine) Validate(cookieValue string) (string, error) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		e.incrementFailures()
		return "", ErrInvalidCookieFormat
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		e.incrementFailures()
		return "", err
	}

	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		e.incrementFailures()
		return "", err
	}

	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write(payload)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		e.incrementFailures()
		return "", ErrInvalidSignature
	}

	return string(payload), nil
}

func (e *CookieEngine) incrementFailures() {
	atomic.AddUint64(&e.FailureCounter[0], 1)
}

func (e *CookieEngine) GetFailureCount() uint64 {
	return atomic.LoadUint64(&e.FailureCounter[0])
}
