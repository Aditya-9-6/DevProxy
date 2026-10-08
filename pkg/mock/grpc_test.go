package mock

import (
	"bytes"
	"testing"
)

func TestGRPCDescriptorEngine(t *testing.T) {
	engine := NewGRPCDescriptorEngine()

	serviceName := "test.v1.TestService"
	fullMethod := "/test.v1.TestService/GetData"
	fdsBytes := []byte("fake-fds-bytes")
	mockResp := []byte("mocked-grpc-response")

	engine.RegisterDescriptor(serviceName, fdsBytes)
	engine.RegisterMock(fullMethod, mockResp)

	// Test Descriptor Lookup
	d, ok := engine.GetDescriptor(serviceName)
	if !ok || !bytes.Equal(d, fdsBytes) {
		t.Errorf("Failed to retrieve registered descriptor")
	}

	// Test Services Listing
	services := engine.ListServices()
	if len(services) != 1 || services[0] != serviceName {
		t.Errorf("Unexpected services list: %v", services)
	}

	// Test Mock Evaluation
	call := &GRPCCallContext{
		ServiceName: "test.v1.TestService",
		MethodName:  "GetData",
		Input:       []byte("request-data"),
	}

	resp, matched := engine.EvaluateMock(call)
	if !matched || !bytes.Equal(resp, mockResp) {
		t.Errorf("Expected mock match, got matched=%v, resp=%s", matched, resp)
	}
}

func TestGRPCPayloadPool(t *testing.T) {
	buf := GRPCPayloadPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString("pooled-buffer")
	if buf.String() != "pooled-buffer" {
		t.Errorf("Unexpected buffer content: %s", buf.String())
	}
	GRPCPayloadPool.Put(buf)
}
