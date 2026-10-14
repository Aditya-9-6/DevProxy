package waf

import (
<<<<<<< HEAD
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
	config      MultipartSanitizerConfig
	activeFiles CacheLinePaddedCounter
	activeDepth CacheLinePaddedCounter
	totalBytes  CacheLinePaddedCounter
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
	err = s.SanitizeStream(reader, boundary, 0)

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
	if currentDepth > s.config.MaxDepth {
		return ErrMultipartDepthExceeded
	}

	s.activeDepth.val.Store(uint64(currentDepth))
	defer func() {
		if currentDepth > 0 && currentDepth-1 >= 0 {
			s.activeDepth.val.Store(uint64(currentDepth - 1))
		} else {
			s.activeDepth.val.Store(0)
		}
	}()

	buf := multipartBufferPool.Get().([]byte)
	defer multipartBufferPool.Put(buf)

	scanner := bufio.NewScanner(r)
	var partBytes int64
	inHeader := true

	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line) + 1)

		currTotal := s.totalBytes.val.Add(uint64(lineLen))
		if s.config.MaxTotalBytes > 0 && int64(currTotal) > s.config.MaxTotalBytes {
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
				files := s.activeFiles.val.Add(1)
				if s.config.MaxFiles > 0 && int(files) > s.config.MaxFiles {
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
=======
	"sync"
	"sync/atomic"
)

// MultipartState tracks boundary nesting depth using atomic counters to prevent injection.
// Uses a fixed-size ring to avoid heap allocations.
type MultipartState struct {
	boundaryDepth  atomic.Int64
	violationCount atomic.Uint64
}

var statePool = sync.Pool{
	New: func() interface{} {
		return &MultipartState{}
	},
}

// NormalizeBoundary checks for nested MIME boundary confusion.
func (s *MultipartState) NormalizeBoundary(boundary []byte) bool {
	// Simple heuristic: nested boundaries often contain recursive patterns.
	// We track depth; if depth > 5, we flag as potential injection.
	depth := s.boundaryDepth.Add(1)
	if depth > 5 {
		s.violationCount.Add(1)
		return false
	}
	return true
}

func (s *MultipartState) Reset() {
	s.boundaryDepth.Store(0)
>>>>>>> origin/main
}
