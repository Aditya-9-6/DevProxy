package quic

import (
	"net/netip"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/security/quicretry"
)

// RetryHandler wraps quicretry TokenManager and ReplayCache for proxy QUIC connections.
type RetryHandler struct {
	tokenManager *quicretry.TokenManager
	replayCache  *quicretry.ReplayCache
}

// NewRetryHandler creates a new RetryHandler.
func NewRetryHandler(maxAge time.Duration) (*RetryHandler, error) {
	tm, err := quicretry.NewTokenManager(maxAge)
	if err != nil {
		return nil, err
	}
	rc := quicretry.NewReplayCache(maxAge)
	return &RetryHandler{
		tokenManager: tm,
		replayCache:  rc,
	}, nil
}

// GenerateRetryToken issues a token for a given client IP and DCID.
func (rh *RetryHandler) GenerateRetryToken(clientIP netip.Addr, originalDCID []byte) ([]byte, error) {
	return rh.tokenManager.GenerateToken(clientIP, originalDCID)
}

// VerifyAndCheckReplay validates the token and ensures it has not been replayed.
func (rh *RetryHandler) VerifyAndCheckReplay(clientIP netip.Addr, token []byte) ([]byte, error) {
	dcid, err := rh.tokenManager.ValidateToken(clientIP, token)
	if err != nil {
		return nil, err
	}

	// Use token string representation as replay identifier
	tokenStr := string(token)
	if !rh.replayCache.CheckAndAdd(tokenStr) {
		return nil, quicretry.ErrInvalidToken
	}

	return dcid, nil
}
