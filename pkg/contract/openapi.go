package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"gopkg.in/yaml.v3"
)

// Parameter defines a parameter in an OpenAPI operation.
type Parameter struct {
	Name     string `json:"name" yaml:"name"`
	In       string `json:"in" yaml:"in"` // "query", "header", "path"
	Required bool   `json:"required" yaml:"required"`
}

// Operation defines an HTTP operation for a path in OpenAPI.
type Operation struct {
	Summary     string              `json:"summary" yaml:"summary"`
	Description string              `json:"description" yaml:"description"`
	Parameters  []Parameter         `json:"parameters" yaml:"parameters"`
	Responses   map[string]Response `json:"responses" yaml:"responses"`
}

// Response defines an expected HTTP response.
type Response struct {
	Description string `json:"description" yaml:"description"`
}

// PathItem defines the operations for an OpenAPI path template.
type PathItem struct {
	Get     *Operation `json:"get" yaml:"get"`
	Post    *Operation `json:"post" yaml:"post"`
	Put     *Operation `json:"put" yaml:"put"`
	Delete  *Operation `json:"delete" yaml:"delete"`
	Patch   *Operation `json:"patch" yaml:"patch"`
	Options *Operation `json:"options" yaml:"options"`
	Head    *Operation `json:"head" yaml:"head"`
}

func (p *PathItem) GetOperation(method string) *Operation {
	switch strings.ToUpper(method) {
	case "GET":
		return p.Get
	case "POST":
		return p.Post
	case "PUT":
		return p.Put
	case "DELETE":
		return p.Delete
	case "PATCH":
		return p.Patch
	case "OPTIONS":
		return p.Options
	case "HEAD":
		return p.Head
	default:
		return nil
	}
}

// OpenAPIDoc represents a parsed OpenAPI v3 or Swagger v2 document.
type OpenAPIDoc struct {
	OpenAPI string `json:"openapi" yaml:"openapi"`
	Swagger string `json:"swagger" yaml:"swagger"`
	Info    struct {
		Title       string `json:"title" yaml:"title"`
		Version     string `json:"version" yaml:"version"`
		Description string `json:"description" yaml:"description"`
	} `json:"info" yaml:"info"`
	Paths map[string]PathItem `json:"paths" yaml:"paths"`
}

type compiledRoute struct {
	pattern  string
	regex    *regexp.Regexp
	pathItem PathItem
}

// Validator validates intercepted HTTP traffic against an OpenAPI / Swagger contract.
type Validator struct {
	mu           sync.RWMutex
	loaded       bool
	doc          *OpenAPIDoc
	routes       []compiledRoute
	rawSpec      string
	totalScanned uint64
	violations   uint64
}

// NewValidator initializes a new OpenAPI contract validator.
func NewValidator() *Validator {
	return &Validator{
		routes: make([]compiledRoute, 0),
	}
}

// LoadSpec parses and compiles an OpenAPI 3.0 or Swagger 2.0 spec (JSON or YAML).
func (v *Validator) LoadSpec(content []byte) error {
	var doc OpenAPIDoc

	// Try JSON first, fallback to YAML
	if err := json.Unmarshal(content, &doc); err != nil {
		if errYaml := yaml.Unmarshal(content, &doc); errYaml != nil {
			return fmt.Errorf("failed to parse spec as JSON (%v) or YAML (%v)", err, errYaml)
		}
	}

	routes := make([]compiledRoute, 0, len(doc.Paths))
	pathParamRegex := regexp.MustCompile(`\{[a-zA-Z0-9_-]+\}`)

	for pathTpl, item := range doc.Paths {
		// Convert OpenAPI path template e.g. /users/{id} to regex ^/users/[^/]+$
		cleanPattern := "^" + pathParamRegex.ReplaceAllString(pathTpl, `[^/]+`) + "$"
		re, err := regexp.Compile(cleanPattern)
		if err != nil {
			continue
		}
		routes = append(routes, compiledRoute{
			pattern:  pathTpl,
			regex:    re,
			pathItem: item,
		})
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.doc = &doc
	v.routes = routes
	v.rawSpec = string(content)
	v.loaded = true
	return nil
}

// ClearSpec unloads current spec.
func (v *Validator) ClearSpec() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.doc = nil
	v.routes = make([]compiledRoute, 0)
	v.rawSpec = ""
	v.loaded = false
}

// IsLoaded reports whether an active specification is loaded.
func (v *Validator) IsLoaded() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.loaded
}

// GetInfo returns metadata about the currently loaded specification.
func (v *Validator) GetInfo() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.loaded || v.doc == nil {
		return map[string]interface{}{
			"loaded": false,
		}
	}

	return map[string]interface{}{
		"loaded":       true,
		"title":        v.doc.Info.Title,
		"version":      v.doc.Info.Version,
		"description":  v.doc.Info.Description,
		"total_paths":  len(v.doc.Paths),
		"routes_count": len(v.routes),
	}
}

// Name implements analysis.Rule
func (v *Validator) Name() string {
	return "OpenAPI Contract Validator"
}

// Evaluate implements analysis.Rule for automated contract drift detection.
func (v *Validator) Evaluate(event *ringbuffer.TrafficEvent) []*analysis.Finding {
	v.mu.RLock()
	if !v.loaded || len(v.routes) == 0 {
		v.mu.RUnlock()
		return nil
	}
	routes := v.routes
	v.mu.RUnlock()

	// Skip non-HTTP methods like CONNECT
	if event.Method == "CONNECT" || event.Method == "" {
		return nil
	}

	var findings []*analysis.Finding
	path := event.Path
	if path == "" && event.URL != "" {
		// Extract path from full URL
		parts := strings.SplitN(event.URL, "://", 2)
		if len(parts) == 2 {
			idx := strings.Index(parts[1], "/")
			if idx != -1 {
				path = parts[1][idx:]
				if qIdx := strings.Index(path, "?"); qIdx != -1 {
					path = path[:qIdx]
				}
			}
		}
	}
	if path == "" {
		path = "/"
	}

	var matchedRoute *compiledRoute
	for i := range routes {
		if routes[i].regex.MatchString(path) {
			matchedRoute = &routes[i]
			break
		}
	}

	now := time.Now()

	// 1. Shadow API Check (Path Undocumented)
	if matchedRoute == nil {
		findings = append(findings, &analysis.Finding{
			ID:          fmt.Sprintf("contract-shadow-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    analysis.SeverityMedium,
			Category:    "CONTRACT_VIOLATION",
			RuleName:    v.Name(),
			Title:       "Undocumented API Endpoint (Shadow API)",
			Description: fmt.Sprintf("Path '%s' is not documented in the loaded OpenAPI specification.", path),
			Evidence:    fmt.Sprintf("Request: %s %s", event.Method, event.URL),
			Location:    "HTTP Request Path",
			Remediation: "Add this endpoint to your OpenAPI/Swagger specification or verify if it is an accidental leak or deprecated route.",
			URL:         event.URL,
			Method:      event.Method,
		})
		return findings
	}

	// 2. HTTP Method Undocumented
	op := matchedRoute.pathItem.GetOperation(event.Method)
	if op == nil {
		findings = append(findings, &analysis.Finding{
			ID:          fmt.Sprintf("contract-method-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    analysis.SeverityHigh,
			Category:    "CONTRACT_VIOLATION",
			RuleName:    v.Name(),
			Title:       fmt.Sprintf("Undocumented HTTP Method '%s'", event.Method),
			Description: fmt.Sprintf("Path '%s' is documented in OpenAPI, but method '%s' is not defined for this endpoint.", matchedRoute.pattern, event.Method),
			Evidence:    fmt.Sprintf("Endpoint %s does not allow %s", matchedRoute.pattern, event.Method),
			Location:    "HTTP Request Method",
			Remediation: "Document the method in OpenAPI schema or restrict allowed HTTP verbs on the server.",
			URL:         event.URL,
			Method:      event.Method,
		})
		return findings
	}

	// 3. Response Status Code Undocumented
	if event.StatusCode > 0 && len(op.Responses) > 0 {
		statusStr := strconv.Itoa(event.StatusCode)
		statusRange := fmt.Sprintf("%dxx", event.StatusCode/100)
		_, hasExact := op.Responses[statusStr]
		_, hasRange := op.Responses[statusRange]
		_, hasDefault := op.Responses["default"]

		if !hasExact && !hasRange && !hasDefault {
			findings = append(findings, &analysis.Finding{
				ID:          fmt.Sprintf("contract-status-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    analysis.SeverityLow,
				Category:    "CONTRACT_VIOLATION",
				RuleName:    v.Name(),
				Title:       fmt.Sprintf("Undocumented Response Status %d", event.StatusCode),
				Description: fmt.Sprintf("Response status %d is not documented for %s %s in the OpenAPI spec.", event.StatusCode, event.Method, matchedRoute.pattern),
				Evidence:    fmt.Sprintf("Returned HTTP %d", event.StatusCode),
				Location:    "HTTP Response Status",
				Remediation: "Document expected error/success status codes in the OpenAPI responses map.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	// 4. Missing Required Parameters (Query & Headers)
	for _, param := range op.Parameters {
		if !param.Required {
			continue
		}
		if param.In == "header" {
			if event.ReqHeaders.Get(param.Name) == "" {
				findings = append(findings, &analysis.Finding{
					ID:          fmt.Sprintf("contract-param-header-%s-%s", event.ID, param.Name),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    analysis.SeverityMedium,
					Category:    "CONTRACT_VIOLATION",
					RuleName:    v.Name(),
					Title:       fmt.Sprintf("Missing Required Header '%s'", param.Name),
					Description: fmt.Sprintf("OpenAPI contract marks header '%s' as required for %s %s.", param.Name, event.Method, matchedRoute.pattern),
					Evidence:    fmt.Sprintf("Missing header: %s", param.Name),
					Location:    "HTTP Request Headers",
					Remediation: fmt.Sprintf("Ensure client includes '%s' header in requests to this endpoint.", param.Name),
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}
	}

	return findings
}
