package proxy

import (
	"context"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

type HandshakeInterceptor struct {
	EntropyEngine *security.EntropyScore
	Threshold     float64
}

func NewHandshakeInterceptor(threshold float64) *HandshakeInterceptor {
	return &HandshakeInterceptor{
		EntropyEngine: &security.EntropyScore{},
		Threshold:     threshold,
	}
}

func (h *HandshakeInterceptor) ShouldTerminate(ctx context.Context, cipherSuites []uint16) bool {
	entropy := security.CalculateShannonEntropy(cipherSuites)
	h.EntropyEngine.UpdateScore(uint64(entropy * 1000))
	return entropy < h.Threshold
}
