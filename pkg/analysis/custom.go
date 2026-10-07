package analysis

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"gopkg.in/yaml.v3"
)

type CustomRuleConfig struct {
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Severity    string `yaml:"severity"`
	Regex       string `yaml:"regex"`
	Description string `yaml:"description"`
	Remediation string `yaml:"remediation"`
	Location    string `yaml:"location"`
}

type CustomRulesDocument struct {
	Rules []CustomRuleConfig `yaml:"rules"`
}

type CustomRule struct {
	cfg   CustomRuleConfig
	regex *regexp.Regexp
}

func NewCustomRule(cfg CustomRuleConfig) (*CustomRule, error) {
	if cfg.Severity == "" {
		cfg.Severity = SeverityHigh
	}
	if cfg.Category == "" {
		cfg.Category = "CUSTOM_RULE"
	}
	if cfg.Location == "" {
		cfg.Location = "all"
	}
	re, err := regexp.Compile(cfg.Regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}
	return &CustomRule{cfg: cfg, regex: re}, nil
}

func (r *CustomRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	loc := strings.ToLower(r.cfg.Location)
	if loc == "all" || loc == "url" {
		if match := r.regex.FindString(event.URL); match != "" {
			findings = append(findings, r.makeFinding(event, "Request URL", match))
		}
	}
	return findings
}

func (r *CustomRule) makeFinding(event *ringbuffer.TrafficEvent, location, evidence string) *Finding {
	return &Finding{
		RequestID: event.ID, Timestamp: event.Timestamp, Severity: strings.ToUpper(r.cfg.Severity),
		Category: r.cfg.Category, RuleName: r.cfg.Name, Title: fmt.Sprintf("%s matched", r.cfg.Name),
		Evidence: evidence, Location: location, Remediation: r.cfg.Remediation,
	}
}

func LoadCustomRulesFromYAML(data []byte) ([]Rule, error) {
	var doc CustomRulesDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var rules []Rule
	for _, cfg := range doc.Rules {
		r, err := NewCustomRule(cfg)
		if err == nil {
			rules = append(rules, r)
		}
	}
	return rules, nil
}
