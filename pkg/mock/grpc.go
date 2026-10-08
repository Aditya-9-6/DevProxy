package mock

import (
	"bytes"
	"fmt"
	"sync"
)

// GRPCPayloadPool provides zero-allocation buffer pooling for gRPC payload serialization.
var GRPCPayloadPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GRPCCallContext represents an incoming gRPC call intercept or mock dispatch.
type GRPCCallContext struct {
	ServiceName string
	MethodName  string
	Input       []byte
	IsStreaming bool
}

// GRPCDescriptorEngine manages protobuf descriptor sets for reflection and mocking.
type GRPCDescriptorEngine struct {
	mu          sync.RWMutex
	descriptors map[string][]byte // map[serviceName]fileDescriptorSetBytes
	mocks       map[string][]byte // map[fullMethodName]mockResponsePayload
}

// NewGRPCDescriptorEngine creates a new gRPC mock and reflection engine.
func NewGRPCDescriptorEngine() *GRPCDescriptorEngine {
	return &GRPCDescriptorEngine{
		descriptors: make(map[string][]byte),
		mocks:       make(map[string][]byte),
	}

}

// RegisterDescriptor registers a serialized FileDescriptorSet for a service.
func (e *GRPCDescriptorEngine) RegisterDescriptor(serviceName string, fds []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.descriptors[serviceName] = fds
}

// RegisterMock registers a static mock payload response for a full gRPC method (e.g. "/package.Service/Method").
func (e *GRPCDescriptorEngine) RegisterMock(fullMethod string, payload []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mocks[fullMethod] = payload
}

// GetDescriptor retrieves the registered FileDescriptorSet for a service.
func (e *GRPCDescriptorEngine) GetDescriptor(serviceName string) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	d, ok := e.descriptors[serviceName]
	return d, ok
}

// GetMock retrieves a registered mock payload for a method.
func (e *GRPCDescriptorEngine) GetMock(fullMethod string) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	m, ok := e.mocks[fullMethod]
	return m, ok
}

// ListServices returns all registered service names.
func (e *GRPCDescriptorEngine) ListServices() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	services := make([]string, 0, len(e.descriptors))
	for s := range e.descriptors {
		services = append(services, s)
	}
	return services
}

// EvaluateMock evaluates an incoming gRPC call against registered mocks.
func (e *GRPCDescriptorEngine) EvaluateMock(call *GRPCCallContext) ([]byte, bool) {
	fullMethod := fmt.Sprintf("/%s/%s", call.ServiceName, call.MethodName)
	return e.GetMock(fullMethod)
}
