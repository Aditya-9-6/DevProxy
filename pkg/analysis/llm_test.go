package analysis

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestLLMRule_OpenAIChatCompletionJSON(t *testing.T) {
	rule := NewLLMRule()

	reqBody := `{"model": "gpt-4o", "messages": [{"role": "user", "content": "Explain quantum computing in simple terms."}]}`
	respBody := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"model": "gpt-4o",
		"choices": [{"message": {"role": "assistant", "content": "Quantum computing uses qubits..."}}],
		"usage": {
			"prompt_tokens": 150,
			"completion_tokens": 250,
			"total_tokens": 400
		}
	}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-llm-1",
		Timestamp:  time.Now(),
		Method:     "POST",
		Host:       "api.openai.com",
		Path:       "/v1/chat/completions",
		URL:        "https://api.openai.com/v1/chat/completions",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(reqBody),
		RespHeaders: http.Header{
			"Content-Type": []string{"application/json"},
		},
		RespBody:   []byte(respBody),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected telemetry finding for OpenAI request, got none")
	}

	foundObs := false
	for _, f := range findings {
		if f.Category == "AI_OBSERVABILITY" && f.Severity == SeverityInfo {
			foundObs = true
			if !containsSubstring(f.Title, "gpt-4o") || !containsSubstring(f.Title, "400 Tokens") {
				t.Errorf("unexpected finding title: %s", f.Title)
			}
		}
	}
	if !foundObs {
		t.Errorf("expected AI_OBSERVABILITY finding for OpenAI call")
	}
}

func TestLLMRule_OpenAISSEStream(t *testing.T) {
	rule := NewLLMRule()

	reqBody := `{"model": "gpt-4o-mini", "stream": true, "messages": [{"role": "user", "content": "Hi"}]}`
	sseResp := "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\" world!\"}}]}\n\n" +
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":40,\"total_tokens\":60}}\n\n" +
		"data: [DONE]\n\n"

	event := &ringbuffer.TrafficEvent{
		ID:         "test-llm-2",
		Timestamp:  time.Now(),
		Method:     "POST",
		Host:       "api.openai.com",
		Path:       "/v1/chat/completions",
		URL:        "https://api.openai.com/v1/chat/completions",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(reqBody),
		RespHeaders: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		RespBody:   []byte(sseResp),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected telemetry finding for OpenAI SSE stream, got none")
	}

	foundObs := false
	for _, f := range findings {
		if f.Category == "AI_OBSERVABILITY" {
			foundObs = true
			if !containsSubstring(f.Evidence, "Tokens: 60") || !containsSubstring(f.Evidence, "Streaming: true") {
				t.Errorf("expected SSE stream evidence with 60 tokens, got: %s", f.Evidence)
			}
		}
	}
	if !foundObs {
		t.Errorf("expected AI_OBSERVABILITY finding for SSE stream")
	}
}

func TestLLMRule_AnthropicMessages(t *testing.T) {
	rule := NewLLMRule()

	reqBody := `{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "Hello Claude"}]}`
	respBody := `{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-5-sonnet-20241022",
		"content": [{"type": "text", "text": "Hello! How can I assist you today?"}],
		"usage": {
			"input_tokens": 85,
			"output_tokens": 115
		}
	}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-llm-3",
		Timestamp:  time.Now(),
		Method:     "POST",
		Host:       "api.anthropic.com",
		Path:       "/v1/messages",
		URL:        "https://api.anthropic.com/v1/messages",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(reqBody),
		RespHeaders: http.Header{
			"Content-Type": []string{"application/json"},
		},
		RespBody:   []byte(respBody),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected telemetry for Anthropic call")
	}

	foundObs := false
	for _, f := range findings {
		if f.Category == "AI_OBSERVABILITY" {
			foundObs = true
			if !containsSubstring(f.Title, "Anthropic") || !containsSubstring(f.Title, "200 Tokens") {
				t.Errorf("unexpected Anthropic finding title: %s", f.Title)
			}
		}
	}
	if !foundObs {
		t.Errorf("expected AI_OBSERVABILITY finding for Anthropic")
	}
}

func TestLLMRule_GeminiContent(t *testing.T) {
	rule := NewLLMRule()

	reqBody := `{"contents": [{"parts": [{"text": "Summarize this article."}]}]}`
	respBody := `{
		"candidates": [{"content": {"parts": [{"text": "Summary here."}]}}],
		"usageMetadata": {
			"promptTokenCount": 210,
			"candidatesTokenCount": 90,
			"totalTokenCount": 300
		}
	}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-llm-4",
		Timestamp:  time.Now(),
		Method:     "POST",
		Host:       "generativelanguage.googleapis.com",
		Path:       "/v1beta/models/gemini-1.5-pro:generateContent",
		URL:        "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro:generateContent",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(reqBody),
		RespHeaders: http.Header{
			"Content-Type": []string{"application/json"},
		},
		RespBody:   []byte(respBody),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected telemetry for Gemini call")
	}

	foundObs := false
	for _, f := range findings {
		if f.Category == "AI_OBSERVABILITY" {
			foundObs = true
			if !containsSubstring(f.Title, "gemini-1.5-pro") || !containsSubstring(f.Title, "300 Tokens") {
				t.Errorf("unexpected Gemini finding title: %s", f.Title)
			}
		}
	}
	if !foundObs {
		t.Errorf("expected AI_OBSERVABILITY finding for Gemini")
	}
}

func TestLLMRule_HighTokenConsumption(t *testing.T) {
	rule := NewLLMRule()

	reqBody := `{"model": "gpt-4o", "messages": [{"role": "user", "content": "Large document"}]}`
	respBody := `{
		"model": "gpt-4o",
		"usage": {
			"prompt_tokens": 7500,
			"completion_tokens": 1500,
			"total_tokens": 9000
		}
	}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-llm-5",
		Timestamp:   time.Now(),
		Method:      "POST",
		Host:        "api.openai.com",
		Path:        "/v1/chat/completions",
		URL:         "https://api.openai.com/v1/chat/completions",
		ReqHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:     []byte(reqBody),
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
		StatusCode:  200,
	}

	findings := rule.Evaluate(event)
	foundWarning := false
	for _, f := range findings {
		if f.Category == "RESOURCE_CONSUMPTION" && f.Severity == SeverityMedium {
			foundWarning = true
			if !containsSubstring(f.Title, "9000 tokens") {
				t.Errorf("expected title to mention 9000 tokens, got: %s", f.Title)
			}
		}
	}
	if !foundWarning {
		t.Errorf("expected high token consumption warning for 9000 tokens")
	}
}

func TestLLMRule_PromptDataLeak(t *testing.T) {
	rule := NewLLMRule()

	// Prompt contains a valid Luhn credit card (4532-0151-1283-0366) and AWS secret key AKIAIOSFODNN7EXAMPLE
	reqBody := `{"model": "gpt-4o", "messages": [
		{"role": "user", "content": "Process customer payment for card 4532-0151-1283-0366 using AWS credential AKIAIOSFODNN7EXAMPLE"}
	]}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-llm-6",
		Timestamp:  time.Now(),
		Method:     "POST",
		Host:       "api.openai.com",
		Path:       "/v1/chat/completions",
		URL:        "https://api.openai.com/v1/chat/completions",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(reqBody),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected data leak findings in prompt, got none")
	}

	foundCard := false
	foundKey := false
	for _, f := range findings {
		if f.Category == "AI_DATA_LEAKAGE" && f.Severity == SeverityCritical {
			if containsSubstring(f.Title, "Credit Card") {
				foundCard = true
			}
			if containsSubstring(f.Title, "Secret / API Key") {
				foundKey = true
			}
		}
	}
	if !foundCard {
		t.Errorf("expected credit card leak finding in prompt")
	}
	if !foundKey {
		t.Errorf("expected AWS secret key leak finding in prompt")
	}
}

func TestLLMRule_LocalOllamaNoLeakAlert(t *testing.T) {
	rule := NewLLMRule()

	// Local Ollama request containing test card
	reqBody := `{"model": "llama3", "messages": [{"role": "user", "content": "Local card 4532-0151-1283-0366"}]}`
	respBody := `{"model": "llama3", "prompt_eval_count": 10, "eval_count": 25}`

	event := &ringbuffer.TrafficEvent{
		ID:          "test-llm-7",
		Timestamp:   time.Now(),
		Method:      "POST",
		Host:        "localhost:11434",
		Path:        "/api/chat",
		URL:         "http://localhost:11434/api/chat",
		ReqHeaders:  http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:     []byte(reqBody),
		RespHeaders: http.Header{"Content-Type": []string{"application/json"}},
		RespBody:    []byte(respBody),
		StatusCode:  200,
	}

	findings := rule.Evaluate(event)
	for _, f := range findings {
		if f.Category == "AI_DATA_LEAKAGE" {
			t.Errorf("local Ollama request should not trigger AI_DATA_LEAKAGE, got finding: %+v", f)
		}
	}
}

func TestEstimateLLMCost(t *testing.T) {
	// GPT-4o: 1,000,000 prompt = $2.50, 1,000,000 completion = $10.00
	cost := EstimateLLMCost("gpt-4o", 1000, 1000)
	expected := (1000.0/1_000_000.0)*2.50 + (1000.0/1_000_000.0)*10.00
	if cost != expected {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}

	// Local llama3 = 0.0
	localCost := EstimateLLMCost("llama3:latest", 5000, 5000)
	if localCost != 0.0 {
		t.Errorf("expected local model cost to be 0, got %f", localCost)
	}
}

func containsSubstring(s, sub string) bool {
	return containsFold(s, sub)
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && len(sub) > 0 && indexFold(s, sub) >= 0))
}

func indexFold(s, sub string) int {
	return indexString(toLower(s), toLower(sub))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexString(s, sub string) int {
	n := len(sub)
	if n == 0 {
		return 0
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return i
		}
	}
	return -1
}
