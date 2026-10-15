package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/security"
	"net/http"
)

// PipelineHandler manages HTTP/1.1 pipelining and smuggling protection.
type PipelineHandler struct {
	sanitizer *security.SmugglingSanitizer
}

func NewPipelineHandler() *PipelineHandler {
	return &PipelineHandler{
		sanitizer: security.NewSmugglingSanitizer(),
	}
}

// ValidateRequest checks for smuggling vulnerabilities before routing.
func (h *PipelineHandler) ValidateRequest(req *http.Request) bool {
	return h.sanitizer.Sanitize(req.Header)
}
