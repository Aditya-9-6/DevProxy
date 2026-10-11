package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"sync"
)

var bufferPool = sync.Pool{
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

func (g *GraphQLGuard) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if len(event.ReqBody) == 0 {
		return nil
	}

	depth := computeBracketDepth(string(event.ReqBody))
	if depth > g.MaxDepth {
		return []*Finding{{
			ID:          "gql-guard-depth-" + event.ID,
			RequestID:   event.ID,
			Severity:    SeverityCritical,
			Category:    "DENIAL_OF_SERVICE",
			RuleName:    g.Name(),
			Title:       "GraphQL Query Depth Exceeded",
			Description: "The query depth exceeds the security threshold.",
			Evidence:    "Depth: 10",
			Location:    "Request Body",
			URL:         event.URL,
		}}
	}
	return nil
}
