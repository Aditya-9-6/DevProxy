package waf

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

func (e *CookieEngine) Validate(cookieValue []byte) ([]byte, error) {
	idx := bytes.IndexByte(cookieValue, '.')
	if idx == -1 {
		e.incrementFailures()
		return nil, ErrInvalidCookieFormat
	}

	payloadPart := cookieValue[:idx]
	sigPart := cookieValue[idx+1:]

	payload := make([]byte, base64.RawURLEncoding.DecodedLen(len(payloadPart)))
	n, err := base64.RawURLEncoding.Decode(payload, payloadPart)
	if err != nil {
		e.incrementFailures()
		return nil, err
	}
	payload = payload[:n]

	sig := make([]byte, hex.DecodedLen(len(sigPart)))
	n, err = hex.Decode(sig, sigPart)
	if err != nil {
		e.incrementFailures()
		return nil, err
	}
	sig = sig[:n]

	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write(payload)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		e.incrementFailures()
		return nil, ErrInvalidSignature
	}

	return payload, nil
}

func (e *CookieEngine) incrementFailures() {
	atomic.AddUint64(&e.FailureCounter[0], 1)
}

func (e *CookieEngine) GetFailureCount() uint64 {
	return atomic.LoadUint64(&e.FailureCounter[0])
}
