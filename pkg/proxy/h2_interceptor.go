package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

type H2Interceptor struct {
	detector *security.ContinuationDetector
}

func NewH2Interceptor() *H2Interceptor {
	return &H2Interceptor{
		detector: security.NewContinuationDetector(),
	}
}

func (h *H2Interceptor) OnContinuationFrame() bool {
	return h.detector.RecordFrame()
}
