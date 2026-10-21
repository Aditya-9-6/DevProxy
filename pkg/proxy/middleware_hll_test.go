package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestWAFIPCardinalityMiddlewareObserveRequest(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		threshold   uint64
		wantCount   uint64
		wantAnomaly bool
	}{
		{name: "malformed address", remoteAddr: "not-an-address", threshold: 1, wantCount: 0, wantAnomaly: false},
		{name: "below threshold", remoteAddr: "192.0.2.10:1234", threshold: 2, wantCount: 1, wantAnomaly: false},
		{name: "threshold reached", remoteAddr: "192.0.2.10:1234", threshold: 1, wantCount: 1, wantAnomaly: true},
		{name: "IPv6 address", remoteAddr: "[2001:db8::10]:1234", threshold: 1, wantCount: 1, wantAnomaly: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middleware := NewWAFIPCardinalityMiddleware(WAFIPCardinalityOptions{
				Threshold: test.threshold,
			})
			request := httptest.NewRequest(http.MethodGet, "http://example.test/resource", nil)
			request.RemoteAddr = test.remoteAddr

			count, anomaly := middleware.ObserveRequest(request)
			if count != test.wantCount {
				t.Fatalf("ObserveRequest() count = %d, want %d", count, test.wantCount)
			}
			if anomaly != test.wantAnomaly {
				t.Fatalf("ObserveRequest() anomaly = %t, want %t", anomaly, test.wantAnomaly)
			}
		})
	}
}

func TestWAFIPCardinalityMiddlewareCustomResolver(t *testing.T) {
	middleware := NewWAFIPCardinalityMiddleware(WAFIPCardinalityOptions{
		Threshold: 1,
		ResolveClientIP: func(*http.Request) netip.Addr {
			return netip.MustParseAddr("203.0.113.8")
		},
	})

	request := httptest.NewRequest(http.MethodGet, "http://example.test/resource", nil)
	request.RemoteAddr = "malformed"

	count, anomaly := middleware.ObserveRequest(request)
	if count != 1 || !anomaly {
		t.Fatalf("ObserveRequest() = (%d, %t), want (1, true)", count, anomaly)
	}
}

func TestWAFIPCardinalityMiddlewareRejectsAndNotifies(t *testing.T) {
	var callbackCount uint64
	middleware := NewWAFIPCardinalityMiddleware(WAFIPCardinalityOptions{
		Threshold:       1,
		RejectAnomalies: true,
		OnAnomaly: func(_ *http.Request, count uint64) {
			callbackCount = count
		},
	})

	nextCalled := false
	handler := middleware.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	request := httptest.NewRequest(http.MethodGet, "http://example.test/resource", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
	if nextCalled {
		t.Fatal("downstream handler was called for a rejected request")
	}
	if callbackCount != 1 {
		t.Fatalf("callback count = %d, want 1", callbackCount)
	}
}

func TestWAFIPCardinalityMiddlewareHotPathAllocations(t *testing.T) {
	middleware := NewWAFIPCardinalityMiddlewareForThreshold(100)
	handler := middleware.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "http://example.test/resource", nil)
	request.RemoteAddr = "192.0.2.30:1234"
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	allocations := testing.AllocsPerRun(1000, func() {
		handler.ServeHTTP(response, request)
	})
	if allocations != 0 {
		t.Fatalf("hot path allocations = %.2f, want 0", allocations)
	}
}
