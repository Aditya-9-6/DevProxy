package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security"
	"net/http"
)

// Filter handles security ingress checks.
type Filter struct {
	repEngine *security.ReputationEngine
}

func NewFilter() *Filter {
	return &Filter{
		repEngine: security.NewReputationEngine(100000),
	}
}

func (f *Filter) ShouldBlock(r *http.Request) bool {
	return f.repEngine.IsBlocked(r.RemoteAddr)
}
