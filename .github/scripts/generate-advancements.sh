#!/usr/bin/env bash
set -e

CHOICE="${1:-all}"

echo "==> Fetching existing issues index in a single fast call..."
EXISTING_TITLES=$(gh issue list --state all --limit 200 --json title --jq '.[].title' 2>/dev/null || true)

create_issue() {
  local title="$1"
  local body="$2"
  local labels="$3"

  echo "==> Checking if issue exists: $title"
  if echo "$EXISTING_TITLES" | grep -Fqx "$title" >/dev/null 2>&1; then
    echo "Issue already exists, skipping."
  else
    echo "Creating issue: $title"
    gh issue create --title "$title" --body "$body" --label "$labels"
    EXISTING_TITLES=$(printf "%s\n%s" "$EXISTING_TITLES" "$title")
  fi
}

# 1. gRPC & Protobuf Decoder
create_grpc_issue() {
  create_issue \
    "feat(advancement): gRPC & Protobuf Stream Decoder in Dashboard" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Unit tests in \`pkg/proxy\` covering gRPC frame parsing." \
    "enhancement,hacktoberfest,help wanted,area/proxy"
}

# 2. Upstream Proxy Chaining
create_proxy_chain_issue() {
  create_issue \
    "feat(advancement): Upstream Proxy Chaining (SOCKS5 & Corporate HTTP Proxy)" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Unit test with a mock upstream proxy verifying chained forwarding." \
    "enhancement,hacktoberfest,good first issue,area/proxy"
}

# 3. HAR Import & Session Playback
create_har_import_issue() {
  create_issue \
    "feat(advancement): External HAR File Import & Session Playback" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Unit tests verifying parsing of sample HAR files." \
    "enhancement,hacktoberfest,good first issue,area/replay"
}

# 4. OpenTelemetry Tracing
create_otel_issue() {
  create_issue \
    "feat(advancement): Distributed OpenTelemetry (OTel) Tracing Propagation" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Conforms to OpenTelemetry HTTP semantic conventions." \
    "enhancement,hacktoberfest,help wanted,area/security"
}

# 5. JA4 TLS Fingerprinting
create_ja4_issue() {
  create_issue \
    "feat(advancement): Client TLS Fingerprinting (JA3 / JA4) Detection" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Displays TLS fingerprint badge in Dashboard." \
    "enhancement,hacktoberfest,help wanted,area/security"
}

# 6. Web Dashboard Light/Dark Mode Switcher
create_ui_theme_issue() {
  create_issue \
    "feat(advancement): High-Contrast Light / Dark Mode Toggle in Web UI" \
    "### 🚀 Feature Proposal & Architecture

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
- [ ] Persists preference across page reloads." \
    "enhancement,hacktoberfest,good first issue,area/dashboard"
}

case "$CHOICE" in
  grpc_protobuf)
    create_grpc_issue
    ;;
  upstream_proxy_chain)
    create_proxy_chain_issue
    ;;
  har_import_playback)
    create_har_import_issue
    ;;
  otel_tracing)
    create_otel_issue
    ;;
  ja4_tls_fingerprint)
    create_ja4_issue
    ;;
  ui_theme_toggle)
    create_ui_theme_issue
    ;;
  all)
    create_grpc_issue
    create_proxy_chain_issue
    create_har_import_issue
    create_otel_issue
    create_ja4_issue
    create_ui_theme_issue
    ;;
  *)
    echo "Unknown choice: $CHOICE"
    exit 1
    ;;
esac

echo "Advancement issue generation complete!"
