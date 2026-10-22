package analysis

import (
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestGraphQLRule_Batching(t *testing.T) {
	rule := NewGraphQLRule()

	// 6 batched queries (limit > 5)
	batchPayload := `[
		{"query": "{ user(id: 1) { name } }"},
		{"query": "{ user(id: 2) { name } }"},
		{"query": "{ user(id: 3) { name } }"},
		{"query": "{ user(id: 4) { name } }"},
		{"query": "{ user(id: 5) { name } }"},
		{"query": "{ user(id: 6) { name } }"}
	]`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-gql-1",
		Timestamp:  time.Now(),
		Method:     "POST",
		URL:        "https://api.example.com/graphql",
		Path:       "/graphql",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(batchPayload),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected batching finding, got none")
	}

	foundBatch := false
	for _, f := range findings {
		if f.Category == "DENIAL_OF_SERVICE" && f.RuleName == rule.Name() {
			foundBatch = true
			break
		}
	}
	if !foundBatch {
		t.Errorf("expected DENIAL_OF_SERVICE finding for batched queries")
	}
}

func TestGraphQLRule_Introspection(t *testing.T) {
	rule := NewGraphQLRule()

	introQuery := `{"query": "query IntrospectionQuery { __schema { queryType { name } } }"}`
	introResp := `{"data": {"__schema": {"queryType": {"name": "Query"}}}}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-gql-2",
		Timestamp:  time.Now(),
		Method:     "POST",
		URL:        "https://api.example.com/graphql",
		Path:       "/graphql",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(introQuery),
		RespBody:   []byte(introResp),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected introspection finding, got none")
	}

	foundIntro := false
	for _, f := range findings {
		if f.Category == "INFORMATION_DISCLOSURE" && f.Title == "GraphQL Introspection Enabled" {
			foundIntro = true
			break
		}
	}
	if !foundIntro {
		t.Errorf("expected GraphQL Introspection Enabled finding")
	}
}

func TestGraphQLRule_QueryDepth(t *testing.T) {
	rule := NewGraphQLRule()

	// Query with depth 8 (limit > 6)
	deepQuery := `{"query": "query { a { b { c { d { e { f { g { h } } } } } } } }"}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-gql-3",
		Timestamp:  time.Now(),
		Method:     "POST",
		URL:        "https://api.example.com/graphql",
		Path:       "/graphql",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(deepQuery),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected query depth finding, got none")
	}

	foundDepth := false
	for _, f := range findings {
		if f.Category == "DENIAL_OF_SERVICE" && f.Title == "Deep GraphQL Query Nesting (Depth 8)" {
			foundDepth = true
			break
		}
	}
	if !foundDepth {
		t.Errorf("expected Deep GraphQL Query Nesting finding, got %+v", findings)
	}
}

func TestGraphQLRule_FieldSuggestionLeak(t *testing.T) {
	rule := NewGraphQLRule()

	query := `{"query": "{ user { usrname } }"}`
	errResp := `{"errors": [{"message": "Cannot query field 'usrname' on type 'User'. Did you mean 'username'?"}]}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-gql-4",
		Timestamp:  time.Now(),
		Method:     "POST",
		URL:        "https://api.example.com/graphql",
		Path:       "/graphql",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(query),
		RespBody:   []byte(errResp),
		StatusCode: 400,
	}

	findings := rule.Evaluate(event)
	if len(findings) == 0 {
		t.Fatalf("expected suggestion leak finding, got none")
	}

	foundSuggest := false
	for _, f := range findings {
		if f.Category == "INFORMATION_DISCLOSURE" && f.Title == "GraphQL Field Suggestion Leak" {
			foundSuggest = true
			break
		}
	}
	if !foundSuggest {
		t.Errorf("expected field suggestion leak finding")
	}
}

func TestGraphQLRule_SafeQuery(t *testing.T) {
	rule := NewGraphQLRule()

	safeQuery := `{"query": "query { user(id: 1) { id name email } }"}`
	safeResp := `{"data": {"user": {"id": "1", "name": "Alice", "email": "alice@example.com"}}}`

	event := &ringbuffer.TrafficEvent{
		ID:         "test-gql-5",
		Timestamp:  time.Now(),
		Method:     "POST",
		URL:        "https://api.example.com/graphql",
		Path:       "/graphql",
		ReqHeaders: http.Header{"Content-Type": []string{"application/json"}},
		ReqBody:    []byte(safeQuery),
		RespBody:   []byte(safeResp),
		StatusCode: 200,
	}

	findings := rule.Evaluate(event)
	if len(findings) != 0 {
		t.Errorf("expected no findings for safe query, got %d", len(findings))
	}
}
