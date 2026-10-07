package dashboard

// ... (existing imports)

// BroadcastEvent publishes a newly processed traffic event and any findings to all active dashboard connections.
func (h *Hub) BroadcastEvent(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) {
	msg := WSMessage{
		Type: "REQUEST",
		Event: map[string]interface{}{
			"id":            event.ID,
			"timestamp":     event.Timestamp,
			"duration_ms":   float64(event.Duration.Nanoseconds()) / 1e6,
			"client_ip":     event.ClientIP,
			"scheme":        event.Scheme,
			"host":          event.Host,
			"method":        event.Method,
			"path":          event.Path,
			"url":           event.URL,
			"status_code":   event.StatusCode,
			"tls":           event.TLS,
			"traceparent":   event.TraceParent,
			"tracestate":    event.TraceState,
			"finding_count": len(findings),
		},
		Findings: findings,
	}
	// ... (rest of function)
}
