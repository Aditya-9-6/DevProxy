param (
    [string]$Choice = "all"
)

function Create-Issue {
    param (
        [string]$Title,
        [string]$Body,
        [string]$Labels
    )

    Write-Host "==> Checking if issue exists: $Title"
    $existing = (gh issue list --search "$Title in:title" --state all --json number --jq 'length')
    if ($existing -and [int]$existing -gt 0) {
        Write-Host "Issue already exists, skipping."
    } else {
        Write-Host "Creating issue: $Title"
        gh issue create --title $Title --body $Body --label $Labels
    }
}

function Create-GrpcIssue {
    $title = "feat(advancement): gRPC & Protobuf Stream Decoder in Dashboard"
    $body = @"
### 🚀 Feature Proposal & Architecture

Currently, DevProxy captures HTTP/1.1, HTTP/2, and SSE streams in raw JSON/text. Modern microservices frequently communicate using **gRPC** over HTTP/2 with binary Protocol Buffers payloads (\`application/grpc\`, \`application/grpc+proto\`).

#### 🎯 Objectives
- Detect \`content-type: application/grpc*\` in the interception layer.
- Parse standard 5-byte gRPC framing headers (1 byte compressed-flag + 4 byte big-endian length prefix).
- Provide a proto reflection or schema-assisted JSON decoder to display human-readable RPC messages in the Web Dashboard.

#### 📂 Target Files
- \`pkg/proxy/proxy.go\`
- \`pkg/analysis/rules.go\`
- \`web/index.html\`

#### ✅ Acceptance Criteria
- [ ] Correctly identifies \`application/grpc\` traffic in the stream list with a \`gRPC\` badge.
- [ ] De-frames message chunks from the gRPC stream.
- [ ] Decodes standard Protobuf fields into JSON representation in the Request/Response viewer.
- [ ] Unit tests in \`pkg/proxy\` covering gRPC frame parsing.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,help wanted,area/proxy"
}

function Create-ProxyChainIssue {
    $title = "feat(advancement): Upstream Proxy Chaining (SOCKS5 & Corporate HTTP Proxy)"
    $body = @"
### 🚀 Feature Proposal & Architecture

Developers inside corporate VPNs or restricted enterprise networks often need their local development tools to route egress traffic through an upstream corporate HTTP or SOCKS5 proxy.

#### 🎯 Objectives
- Add support for an \`-upstream-proxy\` CLI flag (e.g. \`-upstream-proxy=socks5://127.0.0.1:1080\` or \`-upstream-proxy=http://proxy.corp.internal:8080\`).
- Configure \`http.Transport.Proxy\` and SOCKS5 dialer in \`pkg/proxy/proxy.go\` when specified.
- Respect standard environment variables (\`HTTPS_PROXY\`, \`ALL_PROXY\`, \`NO_PROXY\`).

#### 📂 Target Files
- \`cmd/devproxy/main.go\`
- \`pkg/proxy/proxy.go\`
- \`pkg/proxy/proxy_test.go\`

#### ✅ Acceptance Criteria
- [ ] \`-upstream-proxy\` flag parses valid HTTP and SOCKS5 URLs.
- [ ] Forwarding and CONNECT tunnels route through the upstream proxy.
- [ ] Unit test with a mock upstream proxy verifying chained forwarding.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,good first issue,area/proxy"
}

function Create-HarImportIssue {
    $title = "feat(advancement): External HAR File Import & Session Playback"
    $body = @"
### 🚀 Feature Proposal & Architecture

DevProxy currently supports exporting captured traffic to standard **HTTP Archive (.har)** format. Adding the reverse capability—importing external HAR files—will allow developers to replay past production incident sessions or QA recording logs through DevProxy's security rules engine.

#### 🎯 Objectives
- Add \`POST /api/import/har\` REST endpoint in \`pkg/dashboard/server.go\`.
- Parse HAR entries and inject them as synthetic \`TrafficEvent\` records into the ring buffer / SQLite storage.
- Automatically evaluate the imported requests against DevProxy's passive security rules (Secrets, Cookies, PII, LLM, GraphQL).
- Add an \"Import HAR\" button in the Web Dashboard header modal.

#### 📂 Target Files
- \`pkg/storage/har.go\`
- \`pkg/dashboard/server.go\`
- \`web/index.html\`

#### ✅ Acceptance Criteria
- [ ] Valid HAR 1.2 files are parsed into \`TrafficEvent\` structs.
- [ ] Security rules evaluate the imported traffic and populate findings.
- [ ] Web UI displays imported events seamlessly.
- [ ] Unit tests verifying parsing of sample HAR files.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,good first issue,area/replay"
}

function Create-OtelIssue {
    $title = "feat(advancement): Distributed OpenTelemetry (OTel) Tracing Propagation"
    $body = @"
### 🚀 Feature Proposal & Architecture

When debugging microservice architectures, developers need end-to-end trace correlation. Integrating W3C Trace Context and OpenTelemetry will make DevProxy an observability asset for distributed teams.

#### 🎯 Objectives
- Extract W3C \`traceparent\` and \`tracestate\` headers from inbound requests.
- Generate and attach sub-spans for DevProxy processing time (interception, upstream round-trip, ring buffer enqueue).
- Display Trace ID and Span ID links in the Request Inspector panel.
- (Optional) Export OTel traces via OTLP/gRPC to Jaeger or OpenTelemetry Collector.

#### 📂 Target Files
- \`pkg/proxy/proxy.go\`
- \`pkg/ringbuffer/event.go\`
- \`web/index.html\`

#### ✅ Acceptance Criteria
- [ ] Preserves and logs W3C \`traceparent\` in \`TrafficEvent\`.
- [ ] Dashboard displays clickable Trace ID link.
- [ ] Conforms to OpenTelemetry HTTP semantic conventions.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,help wanted,area/security"
}

function Create-Ja4Issue {
    $title = "feat(advancement): Client TLS Fingerprinting (JA3 / JA4) Detection"
    $body = @"
### 🚀 Feature Proposal & Architecture

During TLS handshakes, client Hello messages reveal cipher suites, supported extensions, and elliptic curve formats. Calculating client TLS fingerprints (JA3 / JA4) allows DevProxy to passively detect bot traffic, headless browsers, and User-Agent spoofing.

#### 🎯 Objectives
- Inspect \`tls.ClientHelloInfo\` in the TLS listener / SNI handler before handshake completion.
- Compute the JA3 string (\`SSLVersion,Ciphers,Extensions,EllipticCurves,EllipticCurvePointFormats\`) and its MD5 hash.
- Store the fingerprint in \`TrafficEvent.TLSFingerprint\`.
- Add a security rule flagging User-Agent mismatches (e.g. Chrome User-Agent header with Python/Go TLS fingerprint).

#### 📂 Target Files
- \`pkg/certs/ca.go\`
- \`pkg/proxy/proxy.go\`
- \`pkg/analysis/rules.go\`
- \`web/index.html\`

#### ✅ Acceptance Criteria
- [ ] Correctly computes JA3 hash for incoming HTTPS requests.
- [ ] Flags User-Agent spoofing in the rules engine.
- [ ] Displays TLS fingerprint badge in Dashboard.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,help wanted,area/security"
}

function Create-UiThemeIssue {
    $title = "feat(advancement): High-Contrast Light / Dark Mode Toggle in Web UI"
    $body = @"
### 🚀 Feature Proposal & Architecture

DevProxy currently features a modern cybersecurity dark theme. For accessibility and developers working in high-glare environments, adding a dedicated Light Mode theme switcher with local storage persistence is a great quality-of-life upgrade.

#### 🎯 Objectives
- Implement CSS variables for light theme (\`--bg-dark: #f8fafc\`, \`--bg-card: #ffffff\`, \`--text-main: #0f172a\`, \`--border: #e2e8f0\`).
- Add a Theme Toggle button (☀️ / 🌙) in the dashboard top navigation bar.
- Persist user preference across browser refreshes via \`localStorage.getItem('devproxy_theme')\`.

#### 📂 Target Files
- \`web/index.html\`

#### ✅ Acceptance Criteria
- [ ] Theme toggle smoothly switches between Dark and Light mode.
- [ ] Preserves high contrast and syntax coloring for findings and headers.
- [ ] Persists preference across page reloads.
"@
    Create-Issue -Title $title -Body $body -Labels "enhancement,hacktoberfest,good first issue,area/dashboard"
}

switch ($Choice) {
    "grpc_protobuf"        { Create-GrpcIssue }
    "upstream_proxy_chain" { Create-ProxyChainIssue }
    "har_import_playback"  { Create-HarImportIssue }
    "otel_tracing"         { Create-OtelIssue }
    "ja4_tls_fingerprint"  { Create-Ja4Issue }
    "ui_theme_toggle"      { Create-UiThemeIssue }
    "all" {
        Create-GrpcIssue
        Create-ProxyChainIssue
        Create-HarImportIssue
        Create-OtelIssue
        Create-Ja4Issue
        Create-UiThemeIssue
    }
    Default {
        Write-Error "Unknown choice: $Choice"
        exit 1
    }
}

Write-Host "Advancement issue generation complete!"
