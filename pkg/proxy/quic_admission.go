package proxy

import (
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/protocol/quic"
)

// QUICAdmissionController manages rate limits and state for QUIC connection attempts.
type QUICAdmissionController struct {
	retryHandler *quic.RetryHandler
	activeConns  atomic.Int64
	maxConns     int64
}

// NewQUICAdmissionController initializes admission control for QUIC.
func NewQUICAdmissionController(maxConns int64, tokenMaxAge time.Duration) (*QUICAdmissionController, error) {
	rh, err := quic.NewRetryHandler(tokenMaxAge)
	if err != nil {
		return nil, err
	}
	return &QUICAdmissionController{
		retryHandler: rh,
		maxConns:     maxConns,
	}, nil
}

// ShouldChallenge determines if an incoming handshake needs a Retry token.
func (q *QUICAdmissionController) ShouldChallenge() bool {
	current := q.activeConns.Load()
	return current >= q.maxConns
}

// RetryHandler returns the underlying retry handler.
func (q *QUICAdmissionController) RetryHandler() *quic.RetryHandler {
	return q.retryHandler
}

// IncConns increments active connections.
func (q *QUICAdmissionController) IncConns() {
	q.activeConns.Add(1)
}

// DecConns decrements active connections.
func (q *QUICAdmissionController) DecConns() {
	q.activeConns.Add(-1)
}

// ValidateToken delegates token verification to retry handler.
func (q *QUICAdmissionController) ValidateToken(clientIP netip.Addr, token []byte) ([]byte, error) {
	return q.retryHandler.VerifyAndCheckReplay(clientIP, token)
}
