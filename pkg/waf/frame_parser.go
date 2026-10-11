package waf

import (
	"bytes"
)

type FrameType int

const (
	FrameText FrameType = iota
	FrameBinary
)

func InspectFrame(data []byte, frameType FrameType) bool {
	// Heuristic SQLi/XSS detection
	lower := bytes.ToLower(data)
	if bytes.Contains(lower, []byte("select ")) || bytes.Contains(lower, []byte("union ")) || bytes.Contains(lower, []byte("drop ")) {
		return true
	}
	if bytes.Contains(lower, []byte("<script")) || bytes.Contains(lower, []byte("javascript:")) {
		return true
	}
	return false
}
