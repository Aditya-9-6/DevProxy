package http2flow

// HandleWindowUpdate processes an incoming WINDOW_UPDATE frame.
func (l *Ledger) HandleWindowUpdate(streamID uint32, increment int32) int32 {
	if streamID == 0 {
		return l.AddConnSendWindow(increment)
	}
	l.streamsMu.Lock()
	defer l.streamsMu.Unlock()
	sw, exists := l.streamWindows[streamID]
	if !exists {
		sw = &StreamWindow{sendWindow: 65535, recvWindow: 65535}
		l.streamWindows[streamID] = sw
	}
	sw.sendWindow += increment
	return sw.sendWindow
}
