package waf

import (
	"bytes"
	"encoding/base64"
	"net/url"
)

type FrameType int

const (
	FrameText FrameType = iota
	FrameBinary
)

type Inspector interface {
	Inspect(data []byte) bool
}

type HeuristicInspector struct{}

func (h *HeuristicInspector) Inspect(data []byte) bool {
	decoded, err := url.QueryUnescape(string(data))
	if err != nil {
		decoded = string(data)
	}

	lower := bytes.ToLower([]byte(decoded))
	patterns := [][]byte{
		[]byte("select "), []byte("union "), []byte("drop "),
		[]byte("<script"), []byte("javascript:"),
	}

	for _, p := range patterns {
		if bytes.Contains(lower, p) {
			return true
		}
	}

	if b, err := base64.StdEncoding.DecodeString(decoded); err == nil {
		return h.Inspect(b)
	}

	return false
}

func InspectFrame(data []byte, frameType FrameType) bool {
	inspector := &HeuristicInspector{}
	return inspector.Inspect(data)
}
