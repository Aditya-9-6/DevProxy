package middleware

import (
	"github.com/Aditya-9-6/DevProxy/pkg/telemetry"
	"net/http"
	"time"
)

// TraceRecorder middleware captures request lifecycle events.
type TraceRecorder struct {
	recorder *telemetry.FlightRecorder
}

func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{recorder: telemetry.NewFlightRecorder()}
}

func (t *TraceRecorder) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		if duration > 500*time.Millisecond {
			// Flush logic would trigger here
			_ = t.recorder.Flush()
		}
	})
}
