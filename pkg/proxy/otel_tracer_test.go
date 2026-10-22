package proxy

import (
	"net/http"
	"sync"
	"testing"
)

func TestExtractTraceContext_Valid(t *testing.T) {
	headers := make(http.Header)
	incomingTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	incomingSpanID := "00f067aa0ba902b7"
	headers.Set(HeaderTraceParent, "00-"+incomingTraceID+"-"+incomingSpanID+"-01")
	headers.Set(HeaderTraceState, "rojo=1,congo=2")

	tc := ExtractTraceContext(headers)
	if tc == nil {
		t.Fatal("expected non-nil TraceContext")
	}

	if tc.TraceID != incomingTraceID {
		t.Errorf("expected TraceID %q, got %q", incomingTraceID, tc.TraceID)
	}
	if tc.ParentSpanID != incomingSpanID {
		t.Errorf("expected ParentSpanID %q, got %q", incomingSpanID, tc.ParentSpanID)
	}
	if tc.SpanID == incomingSpanID || len(tc.SpanID) != 16 {
		t.Errorf("expected newly generated child SpanID of 16 chars, got %q", tc.SpanID)
	}
	if tc.TraceState != "rojo=1,congo=2" {
		t.Errorf("expected tracestate to be preserved, got %q", tc.TraceState)
	}
	if !tc.IsSampled {
		t.Errorf("expected IsSampled to be true")
	}

	// Test Injection
	outHeaders := make(http.Header)
	InjectTraceContext(outHeaders, tc)

	outParent := outHeaders.Get(HeaderTraceParent)
	expectedPrefix := "00-" + incomingTraceID + "-" + tc.SpanID + "-01"
	if outParent != expectedPrefix {
		t.Errorf("injected traceparent mismatch: got %q, want %q", outParent, expectedPrefix)
	}
	if outHeaders.Get(HeaderTraceState) != "rojo=1,congo=2" {
		t.Errorf("injected tracestate mismatch: got %q", outHeaders.Get(HeaderTraceState))
	}
}

func TestExtractTraceContext_MissingOrMalformed(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
	}{
		{"empty header", ""},
		{"malformed format", "not-a-trace-parent"},
		{"all zeros trace ID", "00-00000000000000000000000000000000-00f067aa0ba902b7-01"},
		{"all zeros span ID", "00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01"},
		{"too short trace ID", "00-4bf92f3577b34da6-00f067aa0ba902b7-01"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := make(http.Header)
			if tc.raw != "" {
				headers.Set(HeaderTraceParent, tc.raw)
			}

			ctx := ExtractTraceContext(headers)
			if ctx == nil {
				t.Fatal("expected non-nil root TraceContext fallback")
			}
			if len(ctx.TraceID) != 32 {
				t.Errorf("expected 32-char generated TraceID, got %d chars: %q", len(ctx.TraceID), ctx.TraceID)
			}
			if len(ctx.SpanID) != 16 {
				t.Errorf("expected 16-char generated SpanID, got %d chars: %q", len(ctx.SpanID), ctx.SpanID)
			}
			if ctx.ParentSpanID != "" {
				t.Errorf("expected empty ParentSpanID for root span, got %q", ctx.ParentSpanID)
			}
		})
	}
}

func TestTraceContext_ConcurrentSafe(t *testing.T) {
	var wg sync.WaitGroup
	headers := make(http.Header)
	headers.Set(HeaderTraceParent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tc := ExtractTraceContext(headers)
			out := make(http.Header)
			InjectTraceContext(out, tc)
			if out.Get(HeaderTraceParent) == "" {
				t.Errorf("failed concurrent injection")
			}
		}()
	}
	wg.Wait()
}
