package proxy

import (
	"net/http"
)

// Extractor extracts W3C SpanContext from incoming requests or creates a new one if missing.
type Extractor struct {}

func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract pulls traceparent from request headers or generates a fresh span context.
func (e *Extractor) Extract(req *http.Request) SpanContext {
	if req == nil || req.Header == nil {
		return e.NewFreshSpanContext()
	}

	val := req.Header.Get(TraceParentHeader)
	if val == "" {
		return e.NewFreshSpanContext()
	}

	sc, err := ParseTraceParent(val)
	if err != nil {
		return e.NewFreshSpanContext()
	}

	sc.TraceState = req.Header.Get(TraceStateHeader)
	return sc
}

// NewFreshSpanContext generates a brand new root span context.
func (e *Extractor) NewFreshSpanContext() SpanContext {
	return SpanContext{
		Version:    VersionSupported,
		TraceID:    GenerateTraceID(),
		ParentID:   GenerateSpanID(),
		TraceFlags: "01",
		IsValid:    true,
	}
}
