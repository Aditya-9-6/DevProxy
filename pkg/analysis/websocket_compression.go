package analysis

// WebSocketCompressionAnalyzer analyzes WebSocket traffic for compression efficiency and extension usage.
type WebSocketCompressionAnalyzer struct {
	totalMessages int64
	totalCompressed int64
}

// NewWebSocketCompressionAnalyzer creates an analyzer instance.
func NewWebSocketCompressionAnalyzer() *WebSocketCompressionAnalyzer {
	return &WebSocketCompressionAnalyzer{}
}

// RecordFrame records statistics for a processed WebSocket frame.
func (a *WebSocketCompressionAnalyzer) RecordFrame(compressed bool, origSize, compressedSize int) Finding {
	a.totalMessages++
	if compressed {
		a.totalCompressed++
	}
	return Finding{
		ID:          "WS_COMPRESS_METRIC",
		Description: "WebSocket permessage-deflate frame analyzed",
		Severity:    SeverityInfo,
	}
}
