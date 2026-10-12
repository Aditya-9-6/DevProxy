package tls

import (
	"sync"
	"testing"
)

func TestProtocolRouter_ConcurrentRegistrationAndRouting(t *testing.T) {
	router := NewProtocolRouter()
	var wg sync.WaitGroup
	workers := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			proto := "h2"
			if id%2 == 0 {
				proto = "http/1.1"
			}
			router.Register(proto, &mockHandler{})
		}(i)
	}
	wg.Wait()

	protos := router.SupportedProtocols()
	if len(protos) == 0 {
		t.Fatalf("expected supported protocols, got none")
	}

	for _, proto := range protos {
		handler, ok := router.Route(proto)
		if !ok || handler == nil {
			t.Errorf("failed to route protocol %s", proto)
		}
	}
}

func TestALPNNegotiator_GetConfigAndPool(t *testing.T) {
	router := NewProtocolRouter()
	router.Register("h2", &mockHandler{})

	neg := NewALPNNegotiator(router)
	cfg1 := neg.GetConfig()
	if cfg1 == nil {
		t.Fatalf("expected non-nil config")
	}

	neg.PutConfig(cfg1)

	cfg2 := neg.GetConfig()
	if cfg2 == nil {
		t.Fatalf("expected non-nil config from pool")
	}
}
