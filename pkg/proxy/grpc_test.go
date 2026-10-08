package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestHandleGRPCInterceptionMock(t *testing.T) {
	engine := mock.NewGRPCDescriptorEngine()
	fullMethod := "/my.service.Greeter/SayHello"
	engine.RegisterMock(fullMethod, []byte("Hello gRPC mock"))

	ringBuf := ringbuffer.NewRingBuffer(100)

	req := httptest.NewRequest("POST", "/my.service.Greeter/SayHello", bytes.NewBuffer([]byte("req-payload")))
	req.Header.Set("Content-Type", "application/grpc")
	rec := httptest.NewRecorder()

	handled := HandleGRPCInterception(rec, req, engine, ringBuf)
	if !handled {
		t.Errorf("Expected gRPC request to be handled")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/grpc" {
		t.Errorf("Expected Content-Type application/grpc, got %s", ct)
	}
}

func TestHandleGRPCInterceptionNonGRPC(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	handled := HandleGRPCInterception(rec, req, nil, nil)
	if handled {
		t.Errorf("Expected non-gRPC request not to be handled")
	}
}
