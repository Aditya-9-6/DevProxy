package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"sync"
)

var gqlBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4096)
	},
}

type GraphQLGuard struct {
	MaxDepth      int
	MaxComplexity int
}

func NewGraphQLGuard(maxDepth, maxComplexity int) *GraphQLGuard {
	return &GraphQLGuard{MaxDepth: maxDepth, MaxComplexity: maxComplexity}
}

func (g *GraphQLGuard) Name() string {
	return "GraphQL Security Guard"
}

type gqlMetrics struct {
	maxDepth   int
	complexity int
}

func (g *GraphQLGuard) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if len(event.ReqBody) == 0 {
		return nil
	}

	bufPtr := gqlBufferPool.Get().([]byte)
	defer gqlBufferPool.Put(bufPtr)

	metrics := analyzeGraphQL(event.ReqBody)

	var findings []*Finding
	if metrics.maxDepth > g.MaxDepth {
		findings = append(findings, &Finding{
			ID:          "gql-guard-depth-" + event.ID,
			RequestID:   event.ID,
			Severity:    SeverityCritical,
			Category:    "DENIAL_OF_SERVICE",
			RuleName:    g.Name(),
			Title:       "GraphQL Query Depth Exceeded",
			Description: "The query depth exceeds the security threshold.",
			Evidence:    "Depth exceeds max allowed",
			Location:    "Request Body",
			URL:         event.URL,
		})
	}

	if g.MaxComplexity > 0 && metrics.complexity > g.MaxComplexity {
		findings = append(findings, &Finding{
			ID:          "gql-guard-complexity-" + event.ID,
			RequestID:   event.ID,
			Severity:    SeverityCritical,
			Category:    "DENIAL_OF_SERVICE",
			RuleName:    g.Name(),
			Title:       "GraphQL Query Complexity Exceeded",
			Description: "The query complexity exceeds the security threshold.",
			Evidence:    "Complexity exceeds max allowed",
			Location:    "Request Body",
			URL:         event.URL,
		})
	}

	return findings
}

func analyzeGraphQL(body []byte) gqlMetrics {
	maxDepth := 0
	currentDepth := 0
	complexity := 0
	inString := false
	inComment := false
	escaped := false
	hasSelectionSet := false

	for i := 0; i < len(body); i++ {
		b := body[i]

		if inComment {
			if b == '\n' {
				inComment = false
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				inString = false
			}
			continue
		}

		switch b {
		case '#':
			inComment = true
		case '"':
			inString = true
		case '{':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
			hasSelectionSet = true
		case '}':
			if currentDepth > 0 {
				currentDepth--
			}
		case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
			'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
			// Basic complexity scoring increment per field/token identifier
			complexity++
		}
	}

	if !hasSelectionSet && complexity > 0 {
		maxDepth = 1
	}

	return gqlMetrics{
		maxDepth:   maxDepth,
		complexity: complexity,
	}
}
