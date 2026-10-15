package tls

import (
	"context"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

// StapleValidator handles background rotation of OCSP staples.
type StapleValidator struct {
	cache *OCSPCache
}

func NewStapleValidator(cache *OCSPCache) *StapleValidator {
	return &StapleValidator{cache: cache}
}

// ValidateAndRotate checks if a staple is near expiry and triggers refresh.
func (v *StapleValidator) ValidateAndRotate(ctx context.Context, host string) {
	staple := v.cache.Get(host)
	if staple == nil || time.Until(staple.Expiry) < 1*time.Hour {
		// Trigger async refresh logic here
	}
}
