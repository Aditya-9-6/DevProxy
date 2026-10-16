package waf

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"mime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

var (
	ErrMultipartDepthExceeded     = errors.New("waf: multipart boundary nesting depth exceeded")
	ErrMultipartFileLimitExceeded = errors.New("waf: multipart max files exhaustion limit exceeded")
	ErrMultipartSizeExceeded      = errors.New("waf: multipart total body size exceeded")

	multipartBufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	}
)

type CacheLinePaddedCounter struct {
	_   [56]byte
	val atomic.Uint64
	_   [56]byte
}

type MultipartSanitizerConfig struct {
	MaxDepth      int
	MaxFiles      int
	MaxTotalBytes int64
	MaxPartSize   int64
}

func DefaultMultipartConfig() MultipartSanitizerConfig {
	return MultipartSanitizerConfig{
		MaxDepth:      5,
		MaxFiles:      100,
		MaxTotalBytes: 50 * 1024 * 1024,
		MaxPartSize:   10 * 1024 * 1024,
	}
}

type MultipartSanitizer struct {
	config MultipartSanitizerConfig
}

type multipartState struct {
	activeFiles int
	activeDepth int
	totalBytes  int64
}

func NewMultipartSanitizer(cfg MultipartSanitizerConfig) *MultipartSanitizer {
	return &MultipartSanitizer{
		config: cfg,
	}
}

func (s *MultipartSanitizer) Name() string {
	return "HTTP Multipart/Form-Data Security Sanitizer"
}

func (s *MultipartSanitizer) Evaluate(event *ringbuffer.TrafficEvent) []*analysis.Finding {
	contentType := event.ReqHeaders.Get("Content-Type")
	if !isMultipartContentType(contentType) || len(event.ReqBody) == 0 {
		return nil
	}

	_, params, err := mimeParseMediaType(contentType)
	if err != nil {
		return nil
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil
	}

	reader := bytes.NewReader(event.ReqBody)
	state := &multipartState{}
	err = s.SanitizeStreamWithState(reader, boundary, 0, state)

	var findings []*analysis.Finding
	if err != nil {
		severity := analysis.SeverityHigh
		if errors.Is(err, ErrMultipartDepthExceeded) || errors.Is(err, ErrMultipartFileLimitExceeded) {
			severity = analysis.SeverityCritical
		}

		findings = append(findings, &analysis.Finding{
			ID:          "waf-multipart-" + event.ID,
			RequestID:   event.ID,
			Timestamp:   time.Now(),
			Severity:    severity,
			Category:    "DENIAL_OF_SERVICE",
			RuleName:    s.Name(),
			Title:       "Multipart Form-Data Sanitization Violation",
			Description: err.Error(),
			Evidence:    "Boundary nesting or file exhaustion threshold breached",
			Location:    "HTTP Request Body",
			Remediation: "Restrict multipart nesting depth, total payload size, and maximum uploaded files.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	return findings
}

func (s *MultipartSanitizer) SanitizeStream(r io.Reader, boundary string, currentDepth int) error {
	return s.SanitizeStreamWithState(r, boundary, currentDepth, &multipartState{})
}

func (s *MultipartSanitizer) SanitizeStreamWithState(r io.Reader, boundary string, currentDepth int, state *multipartState) error {
	if currentDepth > s.config.MaxDepth {
		return ErrMultipartDepthExceeded
	}

	state.activeDepth = currentDepth

	buf := multipartBufferPool.Get().([]byte)
	defer multipartBufferPool.Put(buf)

	scanner := bufio.NewScanner(r)
	var partBytes int64
	inHeader := true

	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line) + 1)

		state.totalBytes += lineLen
		if s.config.MaxTotalBytes > 0 && state.totalBytes > s.config.MaxTotalBytes {
			return ErrMultipartSizeExceeded
		}

		if bytes.HasPrefix(line, []byte("--"+boundary)) {
			inHeader = true
			partBytes = 0
			if bytes.HasSuffix(line, []byte("--")) {
				return nil
			}
			continue
		}

		if inHeader && len(line) == 0 {
			inHeader = false
			continue
		}

		if !inHeader {
			partBytes += lineLen
			if s.config.MaxPartSize > 0 && partBytes > s.config.MaxPartSize {
				return ErrMultipartSizeExceeded
			}

			if bytes.Contains(line, []byte("Content-Type: multipart/")) {
				state.activeFiles++
				if s.config.MaxFiles > 0 && state.activeFiles > s.config.MaxFiles {
					return ErrMultipartFileLimitExceeded
				}
			}
		}
	}

	return scanner.Err()
}

func isMultipartContentType(ct string) bool {
	return len(ct) >= 19 && (ct[:19] == "multipart/form-data" || ct[:19] == "MULTIPART/FORM-DATA")
}

func mimeParseMediaType(v string) (string, map[string]string, error) {
	return mime.ParseMediaType(v)
}

type MultipartState struct {
	boundaryDepth  atomic.Int64
	violationCount atomic.Uint64
}

func (s *MultipartState) NormalizeBoundary(boundary []byte) bool {
	depth := s.boundaryDepth.Add(1)
	if depth > 5 {
		s.violationCount.Add(1)
		return false
	}
	return true
}

func (s *MultipartState) Reset() {
	s.boundaryDepth.Store(0)
}
