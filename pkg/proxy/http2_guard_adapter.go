package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security/http2guard"
)

type HTTP2GuardAdapter struct {
	guard *http2guard.Guard
}

func NewHTTP2GuardAdapter(cfg http2guard.GuardConfig) *HTTP2GuardAdapter {
	return &HTTP2GuardAdapter{
		guard: http2guard.NewGuard(cfg),
	}
}

func (a *HTTP2GuardAdapter) InspectReset(connID string) bool {
	return a.guard.CheckReset(connID) == http2guard.VerdictAllow
}

func (a *HTTP2GuardAdapter) InspectSettings(connID string) bool {
	return a.guard.CheckSettings(connID) == http2guard.VerdictAllow
}

func (a *HTTP2GuardAdapter) Cleanup(connID string) {
	a.guard.Evict(connID)
}
