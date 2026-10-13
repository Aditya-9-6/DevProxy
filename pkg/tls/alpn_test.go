package tls

import (
	"crypto/tls"
	"sync"
	"testing"
)

// mockHandler implements ProtocolHandler for testing.
type mockHandler struct {
	called bool
}

func (m *mockHandler) Handle(conn *tls.Conn) error {
	m.called = true
	return nil
}

func TestProtocolRouterAndNegotiator(t *testing.T) {
	router := NewProtocolRouter()
	handler := &mockHandler{}

	router.Register("h2", handler)
	router.Register("http/1.1", &mockHandler{})

	protos := router.SupportedProtocols()
	if len(protos) != 2 {
		t.Errorf("expected 2 supported protocols, got %d", len(protos))
	}

	negotiator := NewALPNNegotiator(router)
	cfg := negotiator.GetConfig()
	if cfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	negotiator.PutConfig(cfg)

	// Test concurrent registrations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(val int) {
			defer wg.Done()
			router.Register(string(rune('a'+val%26)), &mockHandler{})
		}(i)
	}
	wg.Wait()
}
