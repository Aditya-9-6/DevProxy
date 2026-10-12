package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"testing"
)

func TestGraphQLGuard_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		maxDepth         int
		maxComplexity    int
		reqBody          string
		expectedFindings int
		expectedCategory string
	}{
		{
			name:             "Valid Depth and Complexity",
			maxDepth:         5,
			maxComplexity:    100,
			reqBody:          "{ user { posts { comments { author { name } } } } }",
			expectedFindings: 0,
		},
		{
			name:             "Exceeded Depth",
			maxDepth:         3,
			maxComplexity:    100,
			reqBody:          "{ a { b { c { d { e } } } } }",
			expectedFindings: 1,
			expectedCategory: "DENIAL_OF_SERVICE",
		},
		{
			name:             "Ignored Braces in Comments and Strings",
			maxDepth:         2,
			maxComplexity:    100,
			reqBody:          "# {ignore this comment}\n{ user { name } } \n " + `query { field(arg: "{string with brace}") }`, // string with brace
			expectedFindings: 0,
		},
		{
			name:             "Empty Body",
			maxDepth:         5,
			maxComplexity:    100,
			reqBody:          "",
			expectedFindings: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			guard := NewGraphQLGuard(tc.maxDepth, tc.maxComplexity)
			event := &ringbuffer.TrafficEvent{
				ID:      "test-" + tc.name,
				ReqBody: []byte(tc.reqBody),
			}

			findings := guard.Evaluate(event)
			if len(findings) != tc.expectedFindings {
				t.Errorf("Expected %d findings, got %d", tc.expectedFindings, len(findings))
			}

			if tc.expectedFindings > 0 && len(findings) > 0 {
				if findings[0].Category != tc.expectedCategory {
					t.Errorf("Expected category %s, got %s", tc.expectedCategory, findings[0].Category)
				}
			}
		})
	}
}
