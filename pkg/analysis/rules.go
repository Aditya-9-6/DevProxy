package analysis

import "github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"

// SecurityRulesEngine handles traffic analysis using a rule-based approach.
type SecurityRulesEngine struct {
	rules []Rule
}

// Rule defines the interface for security analysis modules.
type Rule interface {
	Evaluate(event *ringbuffer.TrafficEvent) []*Finding
}

func NewSecurityRulesEngine() *SecurityRulesEngine {
	return &SecurityRulesEngine{
		rules: make([]Rule, 0),
	}
}

func (e *SecurityRulesEngine) AddRules(rules []Rule) {
	e.rules = append(e.rules, rules...)
}

func (e *SecurityRulesEngine) Analyze(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	for _, rule := range e.rules {
		findings = append(findings, rule.Evaluate(event)...)
	}
	return findings
}
