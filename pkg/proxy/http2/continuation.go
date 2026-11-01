package http2

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security/http2header"
)

type ContinuationGuard struct {
	accountant *http2header.PressureAccountant
}

func NewContinuationGuard(maxSize int64) *ContinuationGuard {
	return &ContinuationGuard{
		accountant: http2header.NewPressureAccountant(maxSize),
	}
}

func (g *ContinuationGuard) CheckChunk(size int64) bool {
	return g.accountant.AddBytes(size)
}
