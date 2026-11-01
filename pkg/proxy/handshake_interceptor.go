package proxy

import (
	"context"
	"errors"
	"net"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

var (
	ErrBotnetDetected = errors.New("proxy: tls handshake rejected due to low entropy botnet fingerprint")
)

// HandshakeInterceptor intercepts TLS connections and applies entropy analysis.
type HandshakeInterceptor struct {
	Threshold uint64
}

// NewHandshakeInterceptor creates a new HandshakeInterceptor with given anomaly threshold.
func NewHandshakeInterceptor(threshold uint64) *HandshakeInterceptor {
	return &HandshakeInterceptor{
		Threshold: threshold,
	}
}

// Intercept inspects incoming raw TLS ClientHello bytes and evaluates anomaly score.
func (hi *HandshakeInterceptor) Intercept(ctx context.Context, conn net.Conn, clientHello []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, anomalyScore := security.AnalyzeTLSClientHello(clientHello)

	var scoreTracker security.CacheLinePaddedScore
	scoreTracker.Store(anomalyScore)

	if scoreTracker.Load() >= hi.Threshold {
		return ErrBotnetDetected
	}

	return nil
}
