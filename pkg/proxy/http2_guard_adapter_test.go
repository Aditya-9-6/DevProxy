package proxy

import (
	"testing"
	"time"
	"github.com/Aditya-9-6/DevProxy/pkg/security/http2guard"
)

func TestHTTP2GuardAdapterFlow(t *testing.T) {
	cfg := http2guard.GuardConfig{
		MaxRapidResets: 1,
		MaxSettings:    1,
		RefillWindow:   time.Second,
		ShardsCount:    2,
	}
	adapter := NewHTTP2GuardAdapter(cfg)
	conn := "adapter-conn"

	if !adapter.InspectReset(conn) {
		t.Error("expected first reset inspection to pass")
	}
	if adapter.InspectReset(conn) {
		t.Error("expected second rapid reset inspection to be blocked")
	}

	if !adapter.InspectSettings(conn) {
		t.Error("expected first settings inspection to pass")
	}
	if adapter.InspectSettings(conn) {
		t.Error("expected second settings inspection to be blocked")
	}

	adapter.Cleanup(conn)
}
