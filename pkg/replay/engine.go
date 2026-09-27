package replay

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

// ReplayOverride specifies modifications to make to a request before re-executing it.
type ReplayOverride struct {
	Method  string              `json:"method,omitempty"`
	URL     string              `json:"url,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    string              `json:"body,omitempty"`
}

// ReplayedResponse contains the outcome of re-transmitting the request.
type ReplayedResponse struct {
	StatusCode int                 `json:"status_code"`
	DurationMs float64             `json:"duration_ms"`
	Headers    map[string][]string `json:"headers"`
	Body       string              `json:"body"`
}

// ResponseDiff outlines semantic variances between original and replayed responses.
type ResponseDiff struct {
	StatusChanged   bool     `json:"status_changed"`
	OldStatus       int      `json:"old_status"`
	NewStatus       int      `json:"new_status"`
	LatencyDiffMs   float64  `json:"latency_diff_ms"`
	HeadersAdded    []string `json:"headers_added"`
	HeadersRemoved  []string `json:"headers_removed"`
	HeadersModified []string `json:"headers_modified"`
	BodyChanged     bool     `json:"body_changed"`
	BodySummary     string   `json:"body_summary"`
}

// ReplayResult encapsulates the entire retest transaction and comparison.
type ReplayResult struct {
	RequestID string                 `json:"request_id"`
	Original  *storage.RequestRecord `json:"original"`
	Replayed  *ReplayedResponse      `json:"replayed"`
	Diff      *ResponseDiff          `json:"diff"`
}

// Replayer executes intercepted requests with optional overrides and computes response diffs.
type Replayer struct {
	client *http.Client
}

// NewReplayer initializes a Replayer with reasonable timeouts and TLS options.
func NewReplayer(timeout time.Duration, insecureTLS bool) *Replayer {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureTLS,
		},
	}
	return &Replayer{
		client: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}

// Replay sends the request (with overrides) and compares it with the original transaction.
func (r *Replayer) Replay(orig *storage.RequestRecord, override *ReplayOverride) (*ReplayResult, error) {
	if orig == nil {
		return nil, fmt.Errorf("original request record cannot be nil")
	}

	method := orig.Method
	targetURL := orig.URL
	bodyStr := orig.ReqBody
	headers := make(http.Header)

	for k, vv := range orig.ReqHeaders {
		for _, v := range vv {
			headers.Add(k, v)
		}
	}

	// Apply overrides if provided
	if override != nil {
		if override.Method != "" {
			method = override.Method
		}
		if override.URL != "" {
			targetURL = override.URL
		}
		if override.Body != "" {
			bodyStr = override.Body
		}
		if override.Headers != nil {
			for k, vv := range override.Headers {
				headers.Del(k)
				for _, v := range vv {
					headers.Add(k, v)
				}
			}
		}
	}

	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = bytes.NewReader([]byte(bodyStr))
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create replay request: %w", err)
	}

	for k, vv := range headers {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("replayed request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read replayed response body: %w", err)
	}

	replayedResp := &ReplayedResponse{
		StatusCode: resp.StatusCode,
		DurationMs: float64(duration.Microseconds()) / 1000.0,
		Headers:    resp.Header,
		Body:       string(respBytes),
	}

	diff := ComputeDiff(orig, replayedResp)

	return &ReplayResult{
		RequestID: orig.ID,
		Original:  orig,
		Replayed:  replayedResp,
		Diff:      diff,
	}, nil
}

// ComputeDiff calculates structural, header, status, and latency changes.
func ComputeDiff(orig *storage.RequestRecord, replayed *ReplayedResponse) *ResponseDiff {
	diff := &ResponseDiff{
		StatusChanged:   orig.StatusCode != replayed.StatusCode,
		OldStatus:       orig.StatusCode,
		NewStatus:       replayed.StatusCode,
		LatencyDiffMs:   replayed.DurationMs - orig.DurationMs,
		HeadersAdded:    make([]string, 0),
		HeadersRemoved:  make([]string, 0),
		HeadersModified: make([]string, 0),
	}

	// Compare Headers
	for k, newVals := range replayed.Headers {
		oldVals, exists := orig.RespHeaders[k]
		if !exists {
			diff.HeadersAdded = append(diff.HeadersAdded, k)
		} else if !reflect.DeepEqual(oldVals, newVals) {
			diff.HeadersModified = append(diff.HeadersModified, k)
		}
	}
	for k := range orig.RespHeaders {
		if _, exists := replayed.Headers[k]; !exists {
			diff.HeadersRemoved = append(diff.HeadersRemoved, k)
		}
	}

	// Compare Body
	diff.BodyChanged = orig.RespBody != replayed.Body
	if diff.BodyChanged {
		// Check if both are JSON
		var origJSON, repJSON interface{}
		err1 := json.Unmarshal([]byte(orig.RespBody), &origJSON)
		err2 := json.Unmarshal([]byte(replayed.Body), &repJSON)
		if err1 == nil && err2 == nil {
			if reflect.DeepEqual(origJSON, repJSON) {
				diff.BodyChanged = false
				diff.BodySummary = "JSON payloads match semantically (whitespace/formatting difference only)."
			} else {
				diff.BodySummary = "JSON content difference detected."
			}
		} else {
			diff.BodySummary = fmt.Sprintf("Body text modified (Original: %d bytes, Replayed: %d bytes).", len(orig.RespBody), len(replayed.Body))
		}
	} else {
		diff.BodySummary = "Response bodies are identical."
	}

	return diff
}
