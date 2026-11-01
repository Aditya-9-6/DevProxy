package quicretry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"net/netip"
	"time"
)

var (
	ErrInvalidToken = errors.New("quicretry: invalid token")
	ErrTokenExpired = errors.New("quicretry: token expired")
)

// TokenManager handles generation and validation of stateless QUIC Retry tokens.
type TokenManager struct {
	gcm    cipher.AEAD
	salt   []byte
	maxAge time.Duration
}

// NewTokenManager creates a new TokenManager with a random AES-GCM key.
func NewTokenManager(maxAge time.Duration) (*TokenManager, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &TokenManager{
		gcm:    gcm,
		salt:   salt,
		maxAge: maxAge,
	}, nil
}

// GenerateToken creates an encrypted token containing client IP, original DCID, and timestamp.
func (tm *TokenManager) GenerateToken(clientIP netip.Addr, originalDCID []byte) ([]byte, error) {
	now := time.Now().UnixNano()
	ipBytes := ipToBytes(clientIP)
	
	// Format: [8 bytes timestamp][1 byte ip_len][16 bytes IP][1+ bytes DCID]
	payloadLen := 8 + 1 + len(ipBytes) + len(originalDCID)
	payload := make([]byte, payloadLen)
	
	// timestamp
	for i := 0; i < 8; i++ {
		payload[i] = byte(now >> uint(56-8*i))
	}
	
	// IP length & IP
	payload[8] = byte(len(ipBytes))
	copy(payload[9:9+len(ipBytes)], ipBytes)
	
	// DCID
	copy(payload[9+len(ipBytes):], originalDCID)

	nonce := make([]byte, tm.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	token := tm.gcm.Seal(nonce, nonce, payload, tm.salt)
	return token, nil
}

// ValidateToken decrypts and validates a stateless retry token.
func (tm *TokenManager) ValidateToken(clientIP netip.Addr, token []byte) ([]byte, error) {
	nonceSize := tm.gcm.NonceSize()
	if len(token) < nonceSize {
		return nil, ErrInvalidToken
	}

	nonce := token[:nonceSize]
	ciphertext := token[nonceSize:]

	payload, err := tm.gcm.Open(nil, nonce, ciphertext, tm.salt)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if len(payload) < 9 {
		return nil, ErrInvalidToken
	}

	var nowNano int64
	for i := 0; i < 8; i++ {
		nowNano = (nowNano << 8) | int64(payload[i])
	}

	issuedAt := time.Unix(0, nowNano)
	if time.Since(issuedAt) > tm.maxAge {
		return nil, ErrTokenExpired
	}

	ipLen := int(payload[8])
	if len(payload) < 9+ipLen {
		return nil, ErrInvalidToken
	}

	ipBytes := payload[9 : 9+ipLen]
	parsedIP := bytesToIP(ipBytes)

	if clientIP != parsedIP {
		return nil, ErrInvalidToken
	}

	originalDCID := make([]byte, len(payload)-(9+ipLen))
	copy(originalDCID, payload[9+ipLen:])

	return originalDCID, nil
}

func ipToBytes(ip netip.Addr) []byte {
	if !ip.IsValid() {
		return nil
	}
	if ip.Is4() {
		arr := ip.As4()
		return arr[:]
	}
	arr := ip.As16()
	return arr[:]
}

func bytesToIP(b []byte) netip.Addr {
	if len(b) == 4 {
		var arr [4]byte
		copy(arr[:], b)
		return netip.AddrFrom4(arr)
	} else if len(b) == 16 {
		var arr [16]byte
		copy(arr[:], b)
		return netip.AddrFrom16(arr)
	}
	return netip.Addr{}
}
