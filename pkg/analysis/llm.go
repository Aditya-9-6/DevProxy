package analysis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// LLMProvider identifies the AI/LLM vendor or engine.
type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "OpenAI"
	ProviderAnthropic LLMProvider = "Anthropic"
	ProviderGemini    LLMProvider = "Google Gemini"
	ProviderOllama    LLMProvider = "Ollama (Local)"
	ProviderGroq      LLMProvider = "Groq"
	ProviderMistral   LLMProvider = "Mistral AI"
	ProviderGeneric   LLMProvider = "Generic LLM"
)

// LLMCallTelemetry encapsulates extracted token counts and cost estimates.
type LLMCallTelemetry struct {
	Provider         LLMProvider `json:"provider"`
	Model            string      `json:"model"`
	PromptTokens     int         `json:"prompt_tokens"`
	CompletionTokens int         `json:"completion_tokens"`
	TotalTokens      int         `json:"total_tokens"`
	EstimatedCostUSD float64     `json:"estimated_cost_usd"`
	IsStreaming      bool        `json:"is_streaming"`
	IsLocal          bool        `json:"is_local"`
}

// LLMRule passively inspects AI/LLM transactions for token usage, costs, and prompt data leaks.
type LLMRule struct {
	acMatcher *AhoCorasickMatcher
}

// NewLLMRule initializes a new AI/LLM traffic inspector rule.
func NewLLMRule() *LLMRule {
	return &LLMRule{
		acMatcher: NewAhoCorasickMatcher(DefaultSecretSignatures()),
	}
}

func (r *LLMRule) Name() string {
	return "AI & LLM Traffic Inspector"
}

func (r *LLMRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	provider, isLLM := identifyLLMProvider(event)
	if !isLLM {
		return nil
	}

	var findings []*Finding
	now := time.Now()

	// 1. Check for Prompt Data Leakage in outbound request body
	if len(event.ReqBody) > 0 {
		promptLeaks := inspectPromptForLeaks(event, provider, r.acMatcher, now)
		if len(promptLeaks) > 0 {
			findings = append(findings, promptLeaks...)
		}
	}

	// 2. Parse Telemetry (Tokens, Model, Cost)
	telemetry := parseLLMTelemetry(event, provider)
	if telemetry != nil && telemetry.TotalTokens > 0 {
		// A. Flag excessive token consumption (>8000 tokens)
		if telemetry.TotalTokens > 8000 {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("llm-tokens-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityMedium,
				Category:    "RESOURCE_CONSUMPTION",
				RuleName:    r.Name(),
				Title:       fmt.Sprintf("High LLM Token Consumption (%d tokens)", telemetry.TotalTokens),
				Description: fmt.Sprintf("AI request consumed %d total tokens (%d prompt, %d completion) with estimated cost of $%.4f. Large prompts increase latency and cloud spend.", telemetry.TotalTokens, telemetry.PromptTokens, telemetry.CompletionTokens, telemetry.EstimatedCostUSD),
				Evidence:    fmt.Sprintf("Model: %s | Prompt: %d | Completion: %d | Cost: $%.5f", telemetry.Model, telemetry.PromptTokens, telemetry.CompletionTokens, telemetry.EstimatedCostUSD),
				Location:    "HTTP Response Body",
				Remediation: "Implement prompt truncation, context window pruning, or prompt caching to control token growth.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		// B. Telemetry Observability Finding
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("llm-telemetry-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityInfo,
			Category:    "AI_OBSERVABILITY",
			RuleName:    r.Name(),
			Title:       fmt.Sprintf("LLM Call: %s (%s) • %d Tokens • Est. $%.5f", telemetry.Model, telemetry.Provider, telemetry.TotalTokens, telemetry.EstimatedCostUSD),
			Description: fmt.Sprintf("Captured %s request using model '%s'. Evaluated %d prompt tokens and %d generated tokens.", telemetry.Provider, telemetry.Model, telemetry.PromptTokens, telemetry.CompletionTokens),
			Evidence:    fmt.Sprintf("Provider: %s | Model: %s | Tokens: %d | Est. Cost: $%.5f | Streaming: %v", telemetry.Provider, telemetry.Model, telemetry.TotalTokens, telemetry.EstimatedCostUSD, telemetry.IsStreaming),
			Location:    "HTTP Transaction",
			Remediation: "Information only: tracked by DevProxy AI Observability Engine.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	return findings
}

func identifyLLMProvider(event *ringbuffer.TrafficEvent) (LLMProvider, bool) {
	host := strings.ToLower(event.Host)
	if h, _, err := net.SplitHostPort(event.Host); err == nil {
		host = strings.ToLower(h)
	}
	path := strings.ToLower(event.Path)

	switch {
	case strings.Contains(host, "api.openai.com") || strings.HasPrefix(path, "/v1/chat/completions") || strings.HasPrefix(path, "/v1/completions"):
		if strings.Contains(host, "groq.com") {
			return ProviderGroq, true
		}
		if strings.Contains(host, "mistral.ai") {
			return ProviderMistral, true
		}
		return ProviderOpenAI, true
	case strings.Contains(host, "api.anthropic.com") || strings.HasPrefix(path, "/v1/messages"):
		return ProviderAnthropic, true
	case strings.Contains(host, "generativelanguage.googleapis.com") || strings.Contains(path, ":generatecontent") || strings.Contains(path, ":streamgeneratecontent"):
		return ProviderGemini, true
	case strings.Contains(host, "api.groq.com"):
		return ProviderGroq, true
	case strings.Contains(host, "api.mistral.ai"):
		return ProviderMistral, true
	case strings.Contains(path, "/api/chat") || strings.Contains(path, "/api/generate") || strings.Contains(event.Host, "11434"):
		return ProviderOllama, true
	default:
		// Check request body for telltale LLM payload structures
		if len(event.ReqBody) > 0 {
			reqStr := string(event.ReqBody)
			if strings.Contains(reqStr, `"messages":`) && strings.Contains(reqStr, `"model":`) {
				return ProviderGeneric, true
			}
		}
		return "", false
	}
}

func inspectPromptForLeaks(event *ringbuffer.TrafficEvent, provider LLMProvider, acMatcher *AhoCorasickMatcher, now time.Time) []*Finding {
	// If provider is entirely local (e.g. Ollama on localhost), data never leaves machine
	host := strings.ToLower(event.Host)
	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || provider == ProviderOllama {
		return nil
	}

	var findings []*Finding
	reqStr := string(event.ReqBody)

	// 1. Credit Card in prompt
	candidates := cardCandidateRegex.FindAllString(reqStr, 3)
	for _, cand := range candidates {
		clean := sanitizeCardDigits(cand)
		if len(clean) >= 13 && len(clean) <= 19 && IsValidLuhn(clean) {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("llm-card-leak-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "AI_DATA_LEAKAGE",
				RuleName:    "AI & LLM Traffic Inspector",
				Title:       fmt.Sprintf("Valid Credit Card Leaked in AI Prompt (%s)", provider),
				Description: fmt.Sprintf("Detected an unmasked credit card number (passing Luhn validation) transmitted in prompt payload to external cloud AI vendor %s. Transmitting PII to external LLMs violates PCI-DSS and privacy regulations.", provider),
				Evidence:    fmt.Sprintf("Card ending in ...%s (Masked: %s)", clean[len(clean)-4:], maskCard(clean)),
				Location:    "HTTP Request Body (Prompt)",
				Remediation: "Sanitize, tokenize, or mask payment card data before sending prompts to third-party AI APIs.",
				URL:         event.URL,
				Method:      event.Method,
			})
			break
		}
	}

	// 2. SSN in prompt
	if ssnMatches := ssnRegex.FindAllString(reqStr, 3); len(ssnMatches) > 0 {
		for _, cand := range ssnMatches {
			if isValidSSN(cand) {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("llm-ssn-leak-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityCritical,
					Category:    "AI_DATA_LEAKAGE",
					RuleName:    "AI & LLM Traffic Inspector",
					Title:       fmt.Sprintf("Social Security Number Leaked in AI Prompt (%s)", provider),
					Description: fmt.Sprintf("Discovered an unmasked US Social Security Number sent in an AI prompt to external vendor %s. This can expose customer identity data to third-party retention and training logs.", provider),
					Evidence:    fmt.Sprintf("SSN format: ***-**-%s", cand[len(cand)-4:]),
					Location:    "HTTP Request Body (Prompt)",
					Remediation: "Scrub Social Security Numbers from RAG context and user prompts before forwarding to cloud models.",
					URL:         event.URL,
					Method:      event.Method,
				})
				break
			}
		}
	}

	// 3. AWS / Secret keys in prompt via Aho-Corasick
	if acMatcher != nil && len(event.ReqBody) > 0 {
		matches := acMatcher.ScanBytes(event.ReqBody)
		for _, m := range matches {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("llm-key-leak-%s-%s", event.ID, m.Signature.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityCritical,
				Category:    "AI_DATA_LEAKAGE",
				RuleName:    "AI & LLM Traffic Inspector",
				Title:       fmt.Sprintf("Active Secret / API Key Dispatched to AI Model (%s)", provider),
				Description: fmt.Sprintf("Discovered secret credential '%s' included in prompt payload to %s. Sensitive API keys should never be fed into LLM prompts or system instructions.", m.Signature.ID, provider),
				Evidence:    fmt.Sprintf("Matched secret %s: %s", m.Signature.Pattern, m.Snippet),
				Location:    "HTTP Request Body (Prompt)",
				Remediation: "Never interpolate environment variables or credential stores into prompt templates.",
				URL:         event.URL,
				Method:      event.Method,
			})
			break
		}
	}

	return findings
}

func parseLLMTelemetry(event *ringbuffer.TrafficEvent, provider LLMProvider) *LLMCallTelemetry {
	telemetry := &LLMCallTelemetry{
		Provider: provider,
		IsLocal:  provider == ProviderOllama || strings.Contains(event.Host, "localhost") || strings.Contains(event.Host, "127.0.0.1"),
	}

	// 1. Extract Model from Request Body or Path
	if len(event.ReqBody) > 0 {
		var reqPayload struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(event.ReqBody, &reqPayload); err == nil && reqPayload.Model != "" {
			telemetry.Model = reqPayload.Model
		}
	}
	if telemetry.Model == "" && provider == ProviderGemini {
		// Gemini URL structure: /v1beta/models/gemini-1.5-flash:generateContent
		if idx := strings.Index(event.Path, "/models/"); idx != -1 {
			sub := event.Path[idx+len("/models/"):]
			if colonIdx := strings.Index(sub, ":"); colonIdx != -1 {
				telemetry.Model = sub[:colonIdx]
			} else {
				telemetry.Model = sub
			}
		}
	}
	if telemetry.Model == "" {
		telemetry.Model = "unknown"
	}

	// 2. Extract Token Counts from Response Body
	if len(event.RespBody) == 0 {
		return telemetry
	}

	respStr := string(event.RespBody)
	isSSE := strings.Contains(respStr, "data:") || strings.Contains(event.RespHeaders.Get("Content-Type"), "text/event-stream")
	telemetry.IsStreaming = isSSE

	if isSSE {
		extractTokensFromSSE(respStr, telemetry)
	} else {
		extractTokensFromJSON(event.RespBody, telemetry)
	}

	// Fallback calculate total tokens if not set
	if telemetry.TotalTokens == 0 && (telemetry.PromptTokens > 0 || telemetry.CompletionTokens > 0) {
		telemetry.TotalTokens = telemetry.PromptTokens + telemetry.CompletionTokens
	}

	// 3. Compute Estimated USD Cost
	telemetry.EstimatedCostUSD = EstimateLLMCost(telemetry.Model, telemetry.PromptTokens, telemetry.CompletionTokens)

	return telemetry
}

func extractTokensFromJSON(respBody []byte, tel *LLMCallTelemetry) {
	// Standard OpenAI / Groq / Mistral format
	var openAIResp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err == nil && openAIResp.Usage.TotalTokens > 0 {
		if tel.Model == "unknown" && openAIResp.Model != "" {
			tel.Model = openAIResp.Model
		}
		tel.PromptTokens = openAIResp.Usage.PromptTokens
		tel.CompletionTokens = openAIResp.Usage.CompletionTokens
		tel.TotalTokens = openAIResp.Usage.TotalTokens
		return
	}

	// Anthropic format
	var anthropicResp struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &anthropicResp); err == nil && (anthropicResp.Usage.InputTokens > 0 || anthropicResp.Usage.OutputTokens > 0) {
		if tel.Model == "unknown" && anthropicResp.Model != "" {
			tel.Model = anthropicResp.Model
		}
		tel.PromptTokens = anthropicResp.Usage.InputTokens
		tel.CompletionTokens = anthropicResp.Usage.OutputTokens
		tel.TotalTokens = tel.PromptTokens + tel.CompletionTokens
		return
	}

	// Gemini format
	var geminiResp struct {
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &geminiResp); err == nil && geminiResp.UsageMetadata.TotalTokenCount > 0 {
		tel.PromptTokens = geminiResp.UsageMetadata.PromptTokenCount
		tel.CompletionTokens = geminiResp.UsageMetadata.CandidatesTokenCount
		tel.TotalTokens = geminiResp.UsageMetadata.TotalTokenCount
		return
	}

	// Ollama format
	var ollamaResp struct {
		Model           string `json:"model"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err == nil && (ollamaResp.PromptEvalCount > 0 || ollamaResp.EvalCount > 0) {
		if tel.Model == "unknown" && ollamaResp.Model != "" {
			tel.Model = ollamaResp.Model
		}
		tel.PromptTokens = ollamaResp.PromptEvalCount
		tel.CompletionTokens = ollamaResp.EvalCount
		tel.TotalTokens = tel.PromptTokens + tel.CompletionTokens
		return
	}
}

func extractTokensFromSSE(respStr string, tel *LLMCallTelemetry) {
	scanner := bufio.NewScanner(strings.NewReader(respStr))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" || payload == "" {
			continue
		}

		var chunk struct {
			Model string `json:"model"`
			Type  string `json:"type"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
				InputTokens      int `json:"input_tokens"`
				OutputTokens     int `json:"output_tokens"`
			} `json:"usage"`
			Message *struct {
				Usage *struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			UsageMetadata *struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}

		if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
			if tel.Model == "unknown" && chunk.Model != "" {
				tel.Model = chunk.Model
			}
			if chunk.Usage != nil {
				if chunk.Usage.PromptTokens > 0 {
					tel.PromptTokens = chunk.Usage.PromptTokens
				}
				if chunk.Usage.CompletionTokens > 0 {
					tel.CompletionTokens = chunk.Usage.CompletionTokens
				}
				if chunk.Usage.TotalTokens > 0 {
					tel.TotalTokens = chunk.Usage.TotalTokens
				}
				if chunk.Usage.InputTokens > 0 {
					tel.PromptTokens = chunk.Usage.InputTokens
				}
				if chunk.Usage.OutputTokens > 0 {
					tel.CompletionTokens = chunk.Usage.OutputTokens
				}
			}
			if chunk.Message != nil && chunk.Message.Usage != nil {
				if chunk.Message.Usage.InputTokens > 0 {
					tel.PromptTokens = chunk.Message.Usage.InputTokens
				}
				if chunk.Message.Usage.OutputTokens > 0 {
					tel.CompletionTokens = chunk.Message.Usage.OutputTokens
				}
			}
			if chunk.UsageMetadata != nil && chunk.UsageMetadata.TotalTokenCount > 0 {
				tel.PromptTokens = chunk.UsageMetadata.PromptTokenCount
				tel.CompletionTokens = chunk.UsageMetadata.CandidatesTokenCount
				tel.TotalTokens = chunk.UsageMetadata.TotalTokenCount
			}
		}
	}
}

// EstimateLLMCost calculates approximate API expenses based on standard 2026 pricing rates.
func EstimateLLMCost(model string, promptTokens, completionTokens int) float64 {
	m := strings.ToLower(model)
	var promptRate, completionRate float64 // Rates in USD per 1,000,000 tokens

	switch {
	case strings.Contains(m, "gpt-4o-mini"):
		promptRate = 0.15
		completionRate = 0.60
	case strings.Contains(m, "gpt-4o"):
		promptRate = 2.50
		completionRate = 10.00
	case strings.Contains(m, "o1-preview"):
		promptRate = 15.00
		completionRate = 60.00
	case strings.Contains(m, "o1-mini"):
		promptRate = 3.00
		completionRate = 12.00
	case strings.Contains(m, "gpt-4-turbo") || strings.Contains(m, "gpt-4-1106"):
		promptRate = 10.00
		completionRate = 30.00
	case strings.Contains(m, "gpt-4"):
		promptRate = 30.00
		completionRate = 60.00
	case strings.Contains(m, "gpt-3.5"):
		promptRate = 0.50
		completionRate = 1.50
	case strings.Contains(m, "claude-3-5-sonnet") || strings.Contains(m, "claude-3.5-sonnet"):
		promptRate = 3.00
		completionRate = 15.00
	case strings.Contains(m, "claude-3-opus"):
		promptRate = 15.00
		completionRate = 75.00
	case strings.Contains(m, "claude-3-haiku"):
		promptRate = 0.25
		completionRate = 1.25
	case strings.Contains(m, "gemini-1.5-pro"):
		promptRate = 3.50
		completionRate = 10.50
	case strings.Contains(m, "gemini-1.5-flash"):
		promptRate = 0.075
		completionRate = 0.30
	case strings.Contains(m, "llama") || strings.Contains(m, "mistral") || strings.Contains(m, "qwen") || strings.Contains(m, "phi"):
		// Open-weight models (Ollama, vLLM) run locally for free
		promptRate = 0.0
		completionRate = 0.0
	default:
		// Generic default rate
		promptRate = 1.00
		completionRate = 3.00
	}

	cost := (float64(promptTokens)/1_000_000.0)*promptRate + (float64(completionTokens)/1_000_000.0)*completionRate
	return cost
}
