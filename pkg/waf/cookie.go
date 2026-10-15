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

func (e *CookieEngine) Validate(cookieValue []byte, dst []byte) (int, error) {
	idx := bytes.IndexByte(cookieValue, '.')
	if idx == -1 {
		e.incrementFailures()
		return 0, ErrInvalidCookieFormat
	}

	payloadPart := cookieValue[:idx]
	sigPart := cookieValue[idx+1:]

	buf := GetBuffer()
	defer PutBuffer(buf)

	n, err := base64.RawURLEncoding.Decode(buf, payloadPart)
	if err != nil {
		e.incrementFailures()
		return 0, err
	}
	payload := buf[:n]

	if len(dst) < len(payload) {
		return 0, errors.New("destination buffer too small")
	}

	sigBuf := GetBuffer()
	defer PutBuffer(sigBuf)
	n, err = hex.Decode(sigBuf, sigPart)
	if err != nil {
		e.incrementFailures()
		return 0, err
	}

	mac := hmac.New(sha256.New, e.secretKey)
	mac.Write(payload)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBuf[:n], expectedSig) {
		e.incrementFailures()
		return 0, ErrInvalidSignature
	}

	copy(dst, payload)
	return len(payload), nil
}

func (e *CookieEngine) incrementFailures() {
	atomic.AddUint64(&e.FailureCounter[0], 1)
}

func (e *CookieEngine) GetFailureCount() uint64 {
	return atomic.LoadUint64(&e.FailureCounter[0])
}
