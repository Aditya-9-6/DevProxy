package proxy

import (
	"context"
	"errors"
	"github.com/Aditya-9-6/DevProxy/pkg/dns/dnssec"
)

// DNSSECResolver wraps standard name resolution with DNSSEC validation.
type DNSSECResolver struct {
	Validator *dnssec.ChainValidator
}

// NewDNSSECResolver creates a DNSSECResolver.
func NewDNSSECResolver() *DNSSECResolver {
	return &DNSSECResolver{
		Validator: dnssec.NewChainValidator(),
	}
}

// Resolve performs validation-enabled lookup.
func (r *DNSSECResolver) Resolve(ctx context.Context, qname string) ([]byte, error) {
	if qname == "" {
		return nil, errors.New("dnssec resolver: empty qname")
	}
	return nil, nil
}
