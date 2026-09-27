package analysis

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// Rule defines the interface for evaluating traffic events.
type Rule interface {
	Name() string
	Evaluate(event *ringbuffer.TrafficEvent) []*Finding
}

// SecurityRulesEngine coordinates and executes security inspection rules on traffic events.
type SecurityRulesEngine struct {
	rules     []Rule
	radixTrie *RadixTrie
}

// NewSecurityRulesEngine creates a fully loaded rules engine with built-in security checks.
func NewSecurityRulesEngine() *SecurityRulesEngine {
	trie := DefaultBlocklistTrie()
	engine := &SecurityRulesEngine{
		radixTrie: trie,
	}

	engine.rules = []Rule{
		NewSecretsRule(),
		NewCookieSecurityRule(),
		NewStackTraceRule(),
		NewSecurityHeadersRule(),
		NewCORSRule(),
		NewInsecureAuthRule(),
		NewIPBlocklistRule(trie),
	}

	return engine
}

// Analyze runs all rules against a traffic event and returns any findings.
func (e *SecurityRulesEngine) Analyze(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	for _, rule := range e.rules {
		if res := rule.Evaluate(event); len(res) > 0 {
			findings = append(findings, res...)
		}
	}
	return findings
}

// --- Rule 1: Secrets & Exposed Credentials ---
type SecretsRule struct {
	patterns []*secretPattern
}

type secretPattern struct {
	name        string
	category    string
	severity    string
	regex       *regexp.Regexp
	description string
	remediation string
}

func NewSecretsRule() *SecretsRule {
	return &SecretsRule{
		patterns: []*secretPattern{
			{
				name:        "AWS Access Key ID",
				category:    "EXPOSED_SECRET",
				severity:    SeverityCritical,
				regex:       regexp.MustCompile(`\b(AKIA[0-9A-Z]{16})\b`),
				description: "Discovered active AWS Access Key ID in payload",
				remediation: "Never hardcode AWS keys. Use IAM roles, environment variables, or AWS Secrets Manager.",
			},
			{
				name:        "GitHub Personal Access Token",
				category:    "EXPOSED_SECRET",
				severity:    SeverityCritical,
				regex:       regexp.MustCompile(`\b(ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9_]{82})\b`),
				description: "Detected GitHub Personal Access Token in traffic",
				remediation: "Revoke the token immediately and inject it securely via secret management tools.",
			},
			{
				name:        "OpenAI API Key",
				category:    "EXPOSED_SECRET",
				severity:    SeverityCritical,
				regex:       regexp.MustCompile(`\b(sk-[a-zA-Z0-9_\-]{20,})\b`),
				description: "Detected OpenAI Secret Key exposed in payload",
				remediation: "Rotate the exposed API key in OpenAI console and store in a secure vault.",
			},
			{
				name:        "Slack Token",
				category:    "EXPOSED_SECRET",
				severity:    SeverityHigh,
				regex:       regexp.MustCompile(`\b(xox[baprs]-[0-9a-zA-Z]{10,48})\b`),
				description: "Detected Slack API token in transmission",
				remediation: "Revoke token and migrate to environment-based secret configuration.",
			},
			{
				name:        "Stripe Live API Key",
				category:    "EXPOSED_SECRET",
				severity:    SeverityCritical,
				regex:       regexp.MustCompile(`\b((?:sk|rk)_live_[0-9a-zA-Z]{24,})\b`),
				description: "Detected Stripe Live Secret Key in transit",
				remediation: "Rotate production Stripe key immediately to prevent unauthorized financial actions.",
			},
			{
				name:        "Private Cryptographic Key",
				category:    "EXPOSED_SECRET",
				severity:    SeverityCritical,
				regex:       regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
				description: "Unencrypted private key detected in payload",
				remediation: "Remove private key from request/response bodies. Never transmit raw private keys.",
			},
			{
				name:        "Google API Key",
				category:    "EXPOSED_SECRET",
				severity:    SeverityHigh,
				regex:       regexp.MustCompile(`\b(AIza[0-9A-Za-z\-_]{35})\b`),
				description: "Detected Google API Key in payload",
				remediation: "Restrict API key scope in Google Cloud Console or move to server-side proxy.",
			},
		},
	}
}

func (r *SecretsRule) Name() string { return "Exposed Secrets & Keys" }

func (r *SecretsRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding

	// Check URL query parameters for leaked tokens
	if strings.Contains(event.URL, "?") {
		lowerURL := strings.ToLower(event.URL)
		sensitiveParams := []string{"api_key=", "apikey=", "token=", "secret=", "password=", "access_token="}
		for _, param := range sensitiveParams {
			if strings.Contains(lowerURL, param) {
				findings = append(findings, &Finding{
					RequestID:   event.ID,
					Timestamp:   event.Timestamp,
					Severity:    SeverityHigh,
					Category:    "CREDENTIAL_LEAK_IN_URI",
					RuleName:    "Sensitive Credential in Query String",
					Title:       fmt.Sprintf("Sensitive parameter '%s' passed in URI", strings.TrimSuffix(param, "=")),
					Description: "Passing secrets in URLs risks disclosure via server access logs, browser history, and Referer headers.",
					Evidence:    event.URL,
					Location:    "Request URL",
					Remediation: "Pass authentication credentials in HTTP request headers (e.g. Authorization: Bearer ...) or POST body.",
					URL:         event.URL,
					Method:      event.Method,
				})
				break
			}
		}
	}

	// Scan Request Body & Response Body for secrets
	targets := []struct {
		name string
		data []byte
	}{
		{"Request Body", event.ReqBody},
		{"Response Body", event.RespBody},
	}

	for _, target := range targets {
		if len(target.data) == 0 {
			continue
		}
		for _, pat := range r.patterns {
			match := pat.regex.Find(target.data)
			if match != nil {
				// Redact part of match for evidence display
				evidence := string(match)
				if len(evidence) > 12 {
					evidence = evidence[:6] + "..." + evidence[len(evidence)-4:]
				}

				findings = append(findings, &Finding{
					RequestID:   event.ID,
					Timestamp:   event.Timestamp,
					Severity:    pat.severity,
					Category:    pat.category,
					RuleName:    pat.name,
					Title:       fmt.Sprintf("%s detected in %s", pat.name, target.name),
					Description: pat.description,
					Evidence:    fmt.Sprintf("Matched pattern in %s: %s", target.name, evidence),
					Location:    target.name,
					Remediation: pat.remediation,
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}
	}

	return findings
}

// --- Rule 2: Insecure Cookie Configuration ---
type CookieSecurityRule struct{}

func NewCookieSecurityRule() *CookieSecurityRule {
	return &CookieSecurityRule{}
}

func (r *CookieSecurityRule) Name() string { return "Cookie Security Flags" }

func (r *CookieSecurityRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding

	cookies := event.RespHeaders["Set-Cookie"]
	for _, cookieStr := range cookies {
		lower := strings.ToLower(cookieStr)
		parts := strings.Split(cookieStr, ";")
		cookieName := "unknown"
		if len(parts) > 0 {
			kv := strings.SplitN(parts[0], "=", 2)
			cookieName = strings.TrimSpace(kv[0])
		}

		isAuthCookie := strings.Contains(strings.ToLower(cookieName), "sess") ||
			strings.Contains(strings.ToLower(cookieName), "auth") ||
			strings.Contains(strings.ToLower(cookieName), "token") ||
			strings.Contains(strings.ToLower(cookieName), "jwt") ||
			strings.Contains(strings.ToLower(cookieName), "id")

		// Missing HttpOnly on potential auth cookie
		if isAuthCookie && !strings.Contains(lower, "httponly") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityHigh,
				Category:    "INSECURE_COOKIE",
				RuleName:    "Missing HttpOnly Flag",
				Title:       fmt.Sprintf("Session cookie '%s' missing HttpOnly flag", cookieName),
				Description: "Cookies without the HttpOnly flag can be stolen by malicious JavaScript in Cross-Site Scripting (XSS) attacks.",
				Evidence:    cookieStr,
				Location:    "Response Header: Set-Cookie",
				Remediation: "Append '; HttpOnly' to the Set-Cookie header.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		// Missing Secure flag on HTTPS
		if event.TLS && !strings.Contains(lower, "secure") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityMedium,
				Category:    "INSECURE_COOKIE",
				RuleName:    "Missing Secure Flag",
				Title:       fmt.Sprintf("Cookie '%s' missing Secure flag over HTTPS", cookieName),
				Description: "Cookies transmitted without the Secure flag may be intercepted in plaintext if redirected over HTTP.",
				Evidence:    cookieStr,
				Location:    "Response Header: Set-Cookie",
				Remediation: "Append '; Secure' to the Set-Cookie header.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		// Lax or missing SameSite
		if !strings.Contains(lower, "samesite") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityLow,
				Category:    "INSECURE_COOKIE",
				RuleName:    "Missing SameSite Attribute",
				Title:       fmt.Sprintf("Cookie '%s' missing SameSite attribute", cookieName),
				Description: "Cookies without SameSite attribute are vulnerable to Cross-Site Request Forgery (CSRF).",
				Evidence:    cookieStr,
				Location:    "Response Header: Set-Cookie",
				Remediation: "Set 'SameSite=Lax' or 'SameSite=Strict' on the cookie.",
				URL:         event.URL,
				Method:      event.Method,
			})
		} else if strings.Contains(lower, "samesite=none") && !strings.Contains(lower, "secure") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityHigh,
				Category:    "INSECURE_COOKIE",
				RuleName:    "SameSite=None Without Secure",
				Title:       fmt.Sprintf("Cookie '%s' specifies SameSite=None without Secure flag", cookieName),
				Description: "Modern browsers reject SameSite=None cookies unless accompanied by the Secure flag.",
				Evidence:    cookieStr,
				Location:    "Response Header: Set-Cookie",
				Remediation: "Ensure 'Secure' is set whenever 'SameSite=None' is declared.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	return findings
}

// --- Rule 3: Verbose Stack Traces in Responses ---
type StackTraceRule struct {
	signatures []*stackSignature
}

type stackSignature struct {
	tech     string
	regex    *regexp.Regexp
	severity string
}

func NewStackTraceRule() *StackTraceRule {
	return &StackTraceRule{
		signatures: []*stackSignature{
			{
				tech:     "Python Traceback",
				regex:    regexp.MustCompile(`(?s)Traceback \(most recent call last\):.*File "[^"]+", line \d+`),
				severity: SeverityHigh,
			},
			{
				tech:     "Java / Spring Stack Trace",
				regex:    regexp.MustCompile(`(?m)^\s*at (?:[a-zA-Z0-9_$]+\.)+[a-zA-Z0-9_$]+\([A-Za-z0-9_$]+\.java:\d+\)`),
				severity: SeverityHigh,
			},
			{
				tech:     "Node.js Error Stack",
				regex:    regexp.MustCompile(`(?m)^\s*at (?:async )?[a-zA-Z0-9_\. <>\/\\:-]+ \([^\)]+:\d+:\d+\)`),
				severity: SeverityHigh,
			},
			{
				tech:     "Go Panic / Runtime Trace",
				regex:    regexp.MustCompile(`(?m)^goroutine \d+ \[[a-z ]+\]:\n[a-zA-Z0-9_\/\.]+\([^\)]*\)\n\t[^\n]+:\d+`),
				severity: SeverityHigh,
			},
			{
				tech:     "PHP Fatal Error / Exception",
				regex:    regexp.MustCompile(`(?i)(Fatal error|Uncaught Exception|Parse error):.*in /.*\.php on line \d+`),
				severity: SeverityHigh,
			},
			{
				tech:     "SQL Database Error Leak",
				regex:    regexp.MustCompile(`(?i)(SQL syntax.*MySQL|PostgreSQL.*ERROR:|ORA-\d{5}|sqlite3\.OperationalError:)`),
				severity: SeverityMedium,
			},
		},
	}
}

func (r *StackTraceRule) Name() string { return "Verbose Stack Trace Disclosure" }

func (r *StackTraceRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if len(event.RespBody) == 0 {
		return nil
	}

	var findings []*Finding
	for _, sig := range r.signatures {
		loc := sig.regex.FindIndex(event.RespBody)
		if loc != nil {
			snippetEnd := loc[1]
			if snippetEnd > loc[0]+200 {
				snippetEnd = loc[0] + 200
			}
			snippet := string(event.RespBody[loc[0]:snippetEnd])

			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    sig.severity,
				Category:    "INFORMATION_DISCLOSURE",
				RuleName:    "Verbose Stack Trace Exposed",
				Title:       fmt.Sprintf("%s detected in HTTP %d response", sig.tech, event.StatusCode),
				Description: "Detailed technical stack traces expose internal file paths, framework versions, and code structure to potential attackers.",
				Evidence:    snippet,
				Location:    "Response Body",
				Remediation: "Catch exceptions gracefully and return sanitized, generic error responses in production/staging environments.",
				URL:         event.URL,
				Method:      event.Method,
			})
			break // one stack trace finding per response is sufficient
		}
	}

	return findings
}

// --- Rule 4: Missing Security Headers & Information Disclosure ---
type SecurityHeadersRule struct{}

func NewSecurityHeadersRule() *SecurityHeadersRule {
	return &SecurityHeadersRule{}
}

func (r *SecurityHeadersRule) Name() string { return "Security Headers Audit" }

func (r *SecurityHeadersRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	// Only audit successful/typical responses
	if event.StatusCode < 200 || event.StatusCode >= 400 {
		return nil
	}

	contentType := event.RespHeaders.Get("Content-Type")
	isHTML := strings.Contains(contentType, "text/html")

	var findings []*Finding

	// Check Server banner leakage
	if server := event.RespHeaders.Get("Server"); server != "" {
		if strings.ContainsAny(server, "/0123456789") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityInfo,
				Category:    "INFO_LEAK",
				RuleName:    "Server Version Banner Leak",
				Title:       fmt.Sprintf("Server header reveals detailed version: %s", server),
				Description: "Disclosing exact web server versions helps attackers target version-specific CVE vulnerabilities.",
				Evidence:    "Server: " + server,
				Location:    "Response Header: Server",
				Remediation: "Configure web server to suppress or genericize the Server response header.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	// Check X-Powered-By
	if poweredBy := event.RespHeaders.Get("X-Powered-By"); poweredBy != "" {
		findings = append(findings, &Finding{
			RequestID:   event.ID,
			Timestamp:   event.Timestamp,
			Severity:    SeverityLow,
			Category:    "INFO_LEAK",
			RuleName:    "X-Powered-By Header Present",
			Title:       fmt.Sprintf("Technology disclosure in X-Powered-By: %s", poweredBy),
			Description: "The X-Powered-By header advertises underlying frameworks (Express, PHP, ASP.NET).",
			Evidence:    "X-Powered-By: " + poweredBy,
			Location:    "Response Header: X-Powered-By",
			Remediation: "Disable X-Powered-By header in application framework settings.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// For HTML pages: check CSP and Clickjacking protections
	if isHTML {
		if event.RespHeaders.Get("Content-Security-Policy") == "" {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityMedium,
				Category:    "MISSING_SECURITY_HEADER",
				RuleName:    "Missing Content-Security-Policy",
				Title:       "Missing Content-Security-Policy (CSP) Header",
				Description: "Content Security Policy prevents Cross-Site Scripting (XSS) and data injection by specifying allowed content sources.",
				Evidence:    "Header missing",
				Location:    "Response Headers",
				Remediation: "Implement a Content-Security-Policy header restricting script-src, object-src, and frame-ancestors.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}

		if event.RespHeaders.Get("X-Frame-Options") == "" && !strings.Contains(event.RespHeaders.Get("Content-Security-Policy"), "frame-ancestors") {
			findings = append(findings, &Finding{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityMedium,
				Category:    "MISSING_SECURITY_HEADER",
				RuleName:    "Missing Clickjacking Defense",
				Title:       "Missing X-Frame-Options or frame-ancestors CSP",
				Description: "Without framing restrictions, application pages can be embedded in malicious iframes for clickjacking attacks.",
				Evidence:    "X-Frame-Options missing",
				Location:    "Response Headers",
				Remediation: "Add 'X-Frame-Options: DENY' or 'SAMEORIGIN', or use CSP 'frame-ancestors'.",
				URL:         event.URL,
				Method:      event.Method,
			})
		}
	}

	// X-Content-Type-Options: nosniff
	if event.RespHeaders.Get("X-Content-Type-Options") != "nosniff" {
		findings = append(findings, &Finding{
			RequestID:   event.ID,
			Timestamp:   event.Timestamp,
			Severity:    SeverityLow,
			Category:    "MISSING_SECURITY_HEADER",
			RuleName:    "Missing X-Content-Type-Options",
			Title:       "Missing X-Content-Type-Options: nosniff",
			Description: "Prevents browsers from MIME-sniffing a response away from the declared content-type.",
			Evidence:    "X-Content-Type-Options missing or not 'nosniff'",
			Location:    "Response Headers",
			Remediation: "Set 'X-Content-Type-Options: nosniff'.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	// HSTS on HTTPS
	if event.TLS && event.RespHeaders.Get("Strict-Transport-Security") == "" {
		findings = append(findings, &Finding{
			RequestID:   event.ID,
			Timestamp:   event.Timestamp,
			Severity:    SeverityMedium,
			Category:    "MISSING_SECURITY_HEADER",
			RuleName:    "Missing Strict-Transport-Security (HSTS)",
			Title:       "Missing HSTS Header on HTTPS response",
			Description: "HTTP Strict Transport Security informs browsers that the site must only be accessed using HTTPS.",
			Evidence:    "Strict-Transport-Security missing",
			Location:    "Response Headers",
			Remediation: "Set 'Strict-Transport-Security: max-age=31536000; includeSubDomains'.",
			URL:         event.URL,
			Method:      event.Method,
		})
	}

	return findings
}

// --- Rule 5: Dangerous CORS Configurations ---
type CORSRule struct{}

func NewCORSRule() *CORSRule     { return &CORSRule{} }
func (r *CORSRule) Name() string { return "CORS Policy Check" }

func (r *CORSRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	allowOrigin := event.RespHeaders.Get("Access-Control-Allow-Origin")
	allowCredentials := event.RespHeaders.Get("Access-Control-Allow-Credentials")

	if allowOrigin == "*" && strings.ToLower(allowCredentials) == "true" {
		return []*Finding{
			{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityHigh,
				Category:    "CORS_MISCONFIGURATION",
				RuleName:    "Overly Permissive CORS with Credentials",
				Title:       "Access-Control-Allow-Origin: * combined with Allow-Credentials: true",
				Description: "Wildcard origin with credentials allowed creates a critical vulnerability where any external website can make authenticated cross-origin requests.",
				Evidence:    fmt.Sprintf("Origin: %s, Credentials: %s", allowOrigin, allowCredentials),
				Location:    "Response Headers",
				Remediation: "Specify an explicit, validated origin whitelist instead of wildcard when credentials are required.",
				URL:         event.URL,
				Method:      event.Method,
			},
		}
	}

	return nil
}

// --- Rule 6: Insecure Authentication over Cleartext HTTP ---
type InsecureAuthRule struct{}

func NewInsecureAuthRule() *InsecureAuthRule { return &InsecureAuthRule{} }
func (r *InsecureAuthRule) Name() string     { return "Insecure Cleartext Authentication" }

func (r *InsecureAuthRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event.TLS {
		return nil
	}

	auth := event.ReqHeaders.Get("Authorization")
	if auth != "" {
		return []*Finding{
			{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityCritical,
				Category:    "INSECURE_TRANSPORT",
				RuleName:    "Cleartext Authentication Credentials",
				Title:       "Authorization header transmitted over unencrypted HTTP",
				Description: "Credentials sent in cleartext HTTP can be intercepted by network sniffers, rogue Wi-Fi access points, or upstream proxies.",
				Evidence:    "Authorization header on non-TLS request",
				Location:    "Request Header: Authorization",
				Remediation: "Always enforce HTTPS/TLS for authenticated routes.",
				URL:         event.URL,
				Method:      event.Method,
			},
		}
	}

	return nil
}

// --- Rule 7: Radix Trie IP/CIDR Blocklist ---
type IPBlocklistRule struct {
	trie *RadixTrie
}

func NewIPBlocklistRule(trie *RadixTrie) *IPBlocklistRule {
	return &IPBlocklistRule{trie: trie}
}

func (r *IPBlocklistRule) Name() string { return "Radix Trie IP/CIDR Matching" }

func (r *IPBlocklistRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	host := event.Host
	if h, _, err := net.SplitHostPort(event.Host); err == nil {
		host = h
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}

	match, found := r.trie.Lookup(ip)
	if !found {
		return nil
	}

	// Flag cloud metadata hits as Critical
	if match.Category == "CLOUD_METADATA" {
		return []*Finding{
			{
				RequestID:   event.ID,
				Timestamp:   event.Timestamp,
				Severity:    SeverityCritical,
				Category:    "METADATA_EXFILTRATION",
				RuleName:    "Cloud Instance Metadata Service Access (169.254.169.254)",
				Title:       "Outbound request to Cloud Metadata Service (SSRF risk)",
				Description: "Application reached 169.254.169.254 (IMDSv1). In production, this can expose IAM roles and cloud credentials to SSRF attacks.",
				Evidence:    fmt.Sprintf("Matched CIDR: %s (%s)", match.CIDR, ip.String()),
				Location:    "Destination IP",
				Remediation: "Enforce IMDSv2 token authentication or disable instance metadata access from containers.",
				URL:         event.URL,
				Method:      event.Method,
			},
		}
	}

	return nil
}
