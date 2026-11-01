package proxy

import (
	"net/http"
	"testing"
)

func TestParseTraceParent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantID  string
	}{
		{
			name:    "valid traceparent",
			input:   "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantErr: false,
			wantID:  "4bf92f3577b34da6a3ce929d0e0e4736",
		},
		{
			name:    "empty header",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid parts count",
			input:   "00-4bf92f3577b34da6a3ce929d0e0e4736-01",
			wantErr: true,
		},
		{
			name:    "unsupported version",
			input:   "01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantErr: true,
		},
		{
			name:    "invalid trace id length",
			input:   "00-4bf92f35-00f067aa0ba902b7-01",
			wantErr: true,
		},
		{
			name:    "zero trace id",
			input:   "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			sc, err := ParseTraceParent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTraceParent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && sc.TraceID != tt.wantID {
				t.Errorf("TraceID = %v, want %v", sc.TraceID, tt.wantID)
			}
		})
	}
}

func TestInjectAndPool(t *testing.T) {
	sc := AcquireSpanContext()
	sc.Version = "00"
	sc.TraceID = GenerateTraceID()
	sc.ParentID = GenerateSpanID()
	sc.TraceFlags = "01"
	sc.TraceState = "rojo=00f067aa0ba902b7"
	sc.IsValid = true

	hdr := make(http.Header)
	Inject(*sc, hdr)

	if hdr.Get(TraceParentHeader) == "" {
		t.Fatal("expected traceparent header to be injected")
	}
	if hdr.Get(TraceStateHeader) != "rojo=00f067aa0ba902b7" {
		t.Fatalf("unexpected tracestate: %s", hdr.Get(TraceStateHeader))
	}

	parsed, err := ParseTraceParent(hdr.Get(TraceParentHeader))
	if err != nil {
		t.Fatalf("failed to parse injected traceparent: %v", err)
	}

	if parsed.TraceID != sc.TraceID {
		t.Errorf("parsed TraceID %s != original %s", parsed.TraceID, sc.TraceID)
	}

	ReleaseSpanContext(sc)
}
