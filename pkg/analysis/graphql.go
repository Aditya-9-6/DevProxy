package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// GraphQLRule passively inspects GraphQL queries and responses for security anti-patterns.
type GraphQLRule struct{}

// NewGraphQLRule creates a new rule instance.
func NewGraphQLRule() *GraphQLRule {
	return &GraphQLRule{}
}

func (r *GraphQLRule) Name() string {
	return "GraphQL Security & Abuse Inspector"
}

func (r *GraphQLRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if len(event.ReqBody) == 0 {
		return nil
	}

	isGraphQL := strings.Contains(strings.ToLower(event.Path), "graphql") ||
		strings.Contains(event.ReqHeaders.Get("Content-Type"), "application/graphql") ||
		bytesContainQuery(event.ReqBody)

	if !isGraphQL {
		return nil
	}

	var findings []*Finding
	now := time.Now()

	// 1. Batching Check
	var batch []map[string]interface{}
	if err := json.Unmarshal(event.ReqBody, &batch); err == nil && len(batch) > 5 {
		findings = append(findings, &Finding{
			ID:          fmt.Sprintf("gql-batch-%s", event.ID),
			RequestID:   event.ID,
			Timestamp:   now,
			Severity:    SeverityMedium,
			Category:    "DENIAL_OF_SERVICE",
			RuleName:    r.Name(),
			Title:       fmt.Sprintf("Excessive GraphQL Query Batching (%d queries)", len(batch)),
			Description: fmt.Sprintf("Client transmitted an array of %d batched GraphQL operations in a single request. Unbounded query batching can be leveraged for Denial of Service (DoS) or brute-force attacks.", len(batch)),
			Evidence:    fmt.Sprintf("Batched count: %d", len(batch)),
			Location:    "HTTP Request Body",
			Remediation: "Implement query batching limits (recommended max 5-10 operations per request) or disable batching in production gateway.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// 2. Query Depth & Introspection Checks
	var single map[string]interface{}
	var queryString string
	if err := json.Unmarshal(event.ReqBody, &single); err == nil {
		if q, ok := single["query"].(string); ok {
			queryString = q
		}
	} else if !strings.HasPrefix(strings.TrimSpace(string(event.ReqBody)), "{") {
		// Raw query format
		queryString = string(event.ReqBody)
	}

	if queryString != "" {
		// A. Introspection Check
		if strings.Contains(queryString, "__schema") || strings.Contains(queryString, "__type") {
			if event.StatusCode == 200 && (strings.Contains(string(event.RespBody), "__schema") || strings.Contains(string(event.RespBody), "types")) {
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("gql-intro-%s", event.ID),
					RequestID:   event.ID,
					Timestamp:   now,
					Severity:    SeverityMedium,
					Category:    "INFORMATION_DISCLOSURE",
					RuleName:    r.Name(),
					Title:       "GraphQL Introspection Enabled",
					Description: "GraphQL schema introspection is enabled and returned schema details. Attackers can map out all queries, mutations, types, and hidden administrative fields.",
					Evidence:    "Query requested __schema/__type and received HTTP 200 with schema metadata.",
					Location:    "HTTP Response Body",
					Remediation: "Disable GraphQL introspection queries in production environments.",
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}

		// B. Query Depth Check
		depth := computeBracketDepth(queryString)
		if depth > 6 {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("gql-depth-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityMedium,
				Category:    "DENIAL_OF_SERVICE",
				RuleName:    r.Name(),
				Title:       fmt.Sprintf("Deep GraphQL Query Nesting (Depth %d)", depth),
				Description: fmt.Sprintf("GraphQL query has a nesting depth of %d levels. Deeply nested queries can exhaust backend database connections and memory.", depth),
				Evidence:    fmt.Sprintf("Computed nesting depth: %d", depth),
				Location:    "HTTP Request Body",
				Remediation: "Configure a GraphQL query depth limiter (e.g., maximum depth of 5-7 levels).",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	// 3. Field Suggestion Leak in Error Responses
	if len(event.RespBody) > 0 {
		respStr := string(event.RespBody)
		if strings.Contains(respStr, "Did you mean") || strings.Contains(respStr, "did you mean") {
			findings = append(findings, &Finding{
				ID:          fmt.Sprintf("gql-suggest-%s", event.ID),
				RequestID:   event.ID,
				Timestamp:   now,
				Severity:    SeverityLow,
				Category:    "INFORMATION_DISCLOSURE",
				RuleName:    r.Name(),
				Title:       "GraphQL Field Suggestion Leak",
				Description: "GraphQL server error contains field suggestions (e.g. 'Did you mean ...?'). This assists attackers in brute-forcing hidden or unlisted schema fields.",
				Evidence:    "Response contains 'Did you mean' suggestion in errors array.",
				Location:    "HTTP Response Body",
				Remediation: "Disable field suggestions or sanitize GraphQL error messages in production.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	return findings
}

func bytesContainQuery(data []byte) bool {
	s := string(data)
	return strings.Contains(s, `"query"`) || strings.Contains(s, "query ") || strings.Contains(s, "mutation ")
}

func computeBracketDepth(query string) int {
	maxDepth := 0
	currentDepth := 0
	inString := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '"' && (i == 0 || query[i-1] != '\\') {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		if ch == '{' {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		} else if ch == '}' {
			if currentDepth > 0 {
				currentDepth--
			}
		}
	}

	return maxDepth
}
