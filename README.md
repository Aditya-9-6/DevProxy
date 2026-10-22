# DevProxy: Zero-Latency Development Security Proxy

[![CI](https://github.com/Aditya-9-6/DevProxy/actions/workflows/pr-validation.yml/badge.svg)](https://github.com/Aditya-9-6/DevProxy/actions/workflows/pr-validation.yml)
[![DeepSource](https://app.deepsource.com/gh/Aditya-9-6/DevProxy.svg/?label=active+issues&show_trend=true)](https://app.deepsource.com/gh/Aditya-9-6/DevProxy/)
[![DeepSource](https://app.deepsource.com/gh/Aditya-9-6/DevProxy.svg/?label=resolved+issues&show_trend=true)](https://app.deepsource.com/gh/Aditya-9-6/DevProxy/)
[![Go Report Card](https://goreportcard.com/badge/github.com/Aditya-9-6/DevProxy)](https://goreportcard.com/report/github.com/Aditya-9-6/DevProxy)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#-contributors)

> Completely decoupled, asynchronous passive security proxy that inspects web traffic for vulnerabilities with **zero added latency** to the developer's data path.

---

## ⚡ Architectural Blueprint

Traditional security scanning proxies insert synchronous inspection into the request/response cycle, adding hundreds of milliseconds of latency and breaking local developer workflows.

**DevProxy** resolves this by completely **decoupling the Primary Data Path (traffic routing) from the Analysis Path (vulnerability scanning)**. The proxy forwards packets immediately at line-rate and hands off a cloned byte stream to a lock-free ring buffer for asynchronous evaluation.

```
       [ Client Application / Curl / Browser ]
                          │
                          ▼
            ┌───────────────────────────┐
            │   THE INTERCEPTION LAYER  │
            │     (Primary Data Path)   │
            │  • Asynchronous I/O       │
            │  • Dynamic TLS Bumping    │
            │  • eBPF / Transparent     │
            └─────────────┬─────────────┘
                          │
      ┌───────────────────┴───────────────────┐
      │ Primary Path (Immediate Forwarding)   │ Cloned Byte Stream
      ▼                                       ▼
┌──────────────┐                  ┌───────────────────────────────┐
│ Target Host  │                  │     LOCK-FREE RING BUFFER     │
│  (Upstream)  │                  │  (Bounded, Non-Blocking Push) │
└──────────────┘                  └───────────────┬───────────────┘
                                                  │
                                                  ▼
                                  ┌───────────────────────────────┐
                                  │   ANALYSIS WORKER THREADS     │
                                  │ • Single-Pass Parsing         │
                                  │ • Bitwise Radix Trie (CIDR)   │
                                  │ • Secrets & Credential Match  │
                                  │ • Cookie Security Auditor     │
                                  │ • Stack Trace Classifier      │
                                  └───────────────┬───────────────┘
                                                  │
                                                  ▼
                                  ┌───────────────────────────────┐
                                  │      LOCAL DASHBOARD & DB     │
                                  │ • Ephemeral In-Memory SQLite  │
                                  │ • WebSocket Event Broadcaster │
                                  │ • Reactive Dark-Mode Web UI   │
                                  └───────────────────────────────┘
```

---

## 🧩 Core Innovations

### 1. The Interception Layer (The Data Path)
* **Zero-Blocking Forwarding:** The data path streams bytes directly between client and upstream server using non-blocking goroutines and asynchronous connection multiplexing. Network hop overhead is `< 1ms`.
* **On-the-Fly TLS Bumping:** Generates a persistent local Root CA (`devproxy-ca.crt`) upon first launch. During HTTPS `CONNECT` tunnels, leaf certificates are dynamically minted and signed using fast **ECDSA P-256** with an in-memory LRU cache, enabling instantaneous TLS handshakes.
* **Transparent Redirection Support:** Provides an architectural hook for Linux `cgroup` eBPF socket filters (`sock_ops`) and `iptables` redirection to silently intercept container traffic without modifying environment variables.

### 2. The Zero-Allocation Analysis Engine
* **Lock-Free Circular Ring Buffer:** As traffic is streamed, request and response pairs are pushed into a bounded ring buffer with atomic cursor indices. If the security analyzer encounters a CPU spike, the buffer gracefully drops analysis frames rather than backpressuring the client.
* **High-Speed Rules Engine:**
  - 🔑 **API Secret Leak Detection:** Detects exposed AWS Access Keys (`AKIA...`), GitHub Personal Access Tokens (`ghp_...`, `github_pat_...`), OpenAI API keys (`sk-...`), Slack tokens (`xox[baprs]-...`), Google API Keys, Stripe Live Keys, and unencrypted private keys.
  - 🍪 **Cookie Security Flags:** Flags session and authentication cookies missing `HttpOnly`, `Secure`, or `SameSite` attributes, as well as dangerous `SameSite=None` without `Secure`.
  - 💥 **Verbose Stack Trace Disclosure:** Identifies detailed technical error traces from Java/Spring, Python tracebacks, Node.js V8 call stacks, Go runtime panics, PHP fatal errors, and SQL syntax disclosures.
  - 🌲 **Bitwise Radix Trie for IP/CIDR Matching:** $O(1)$ zero-allocation trie for detecting dangerous outbound destinations, such as Cloud Instance Metadata Service (`169.254.169.254` SSRF risk) and RFC 1918 internal networks.
  - 🛡️ **Security Headers & CORS Audit:** Detects missing `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `Strict-Transport-Security`, clickjacking exposure, and overly permissive CORS (`*` with credentials).

### 3. Developer Superpowers (Productivity & Resilience)
* 🎭 **Map Local & Map Remote Mocking:** Intercept matched requests with local files, fixtures, or custom responses without hitting backend APIs. Rewrite upstream targets (e.g. forward staging to `localhost:3000`) for seamless debugging.
* ⚡ **Chaos Engineering & Resilience Simulation:** Inject deterministic or probabilistic latency (with configurable jitter) and HTTP error faults (500, 502, 503) directly in the proxy to test client retry logic and circuit breakers.
* 🔍 **Aho-Corasick Streaming Secret Scanner:** Pure-Go $O(N + M)$ multi-pattern automaton scanning live HTTP request/response payloads for leaked credentials (AWS, GitHub, Slack, OpenAI, Stripe, private keys) with zero backtracking.
* 🛡️ **JWT Security Linter & Passive Inspector:** Automatically detects Bearer tokens, cookies, and JSON payloads; flags dangerous `alg: "none"` algorithms, expired tokens, excessive lifetimes (>1 year), and unencrypted sensitive PII or credentials in claims.
* 📜 **OpenAPI Contract Drift & Shadow API Detection:** Passively validates intercepted HTTP traffic against OpenAPI 3.0 / Swagger 2.0 schemas. Instantly flags undocumented endpoints (Shadow APIs), invalid HTTP verbs, and undocumented response codes.

### 4. Ephemeral Storage & Real-Time Dashboard
* **In-Memory SQLite Datastore:** Uses pure-Go SQLite (`modernc.org/sqlite`) running in-memory with shared cache mode. No CGO or external C compiler needed on Windows, macOS, or Linux.
* **WebSocket Stream:** Embedded server pushes new traffic events and flagged vulnerabilities over `/ws` with sub-millisecond dispatch.
* **Embedded Dark-Mode Dashboard:** A responsive, self-contained web console embedded directly into the binary with `go:embed`. Features live search, status filtering, one-click Root CA download, HAR export, Copy as cURL, JWT inspector, Mocks modal, and OpenAPI validator.

---

## 🚀 Quick Start

### 1. Installation & Build

```bash
# Clone the repository
git clone https://github.com/Aditya-9-6/DevProxy.git
cd DevProxy

# Build the single self-contained binary
go build -o devproxy ./cmd/devproxy
```

### 2. Launching DevProxy

```bash
./devproxy
```

Default listening ports:
* **Proxy Engine:** `http://127.0.0.1:8080`
* **Web Dashboard:** `http://127.0.0.1:8081`

### 3. Configuring Your Application / CLI

Run curl or any developer tool through the proxy:

```bash
# Plain HTTP:
curl -x http://localhost:8080 http://httpbin.org/get

# HTTPS (using the auto-generated Root CA):
curl -x http://localhost:8080 --cacert ~/.devproxy/devproxy-ca.crt https://httpbin.org/get

# Or set global environment variables:
export HTTP_PROXY="http://127.0.0.1:8080"
export HTTPS_PROXY="http://127.0.0.1:8080"
```

Open `http://localhost:8081` in your browser to inspect live traffic and real-time security alerts.

---

## 🛠️ CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-port` | `8080` | Port for the HTTP/HTTPS proxy engine |
| `-web-port` | `8081` | Port for the web dashboard and REST/WebSocket API |
| `-buffer-size` | `16384` | Capacity of the lock-free circular ring buffer |
| `-workers` | `4` | Number of concurrent background analysis workers |
| `-rules` | `devproxy.yaml` | Path to custom YAML security rules file |
| `-openapi` | `openapi.yaml` | Path to OpenAPI 3.0 / Swagger schema for contract validation |
| `-ca-cert` | `~/.devproxy/devproxy-ca.crt` | Custom Root CA certificate path |
| `-ca-key` | `~/.devproxy/devproxy-ca.key` | Custom Root CA private key path |
| `-ebpf` | `false` | Show transparent eBPF / container redirection guide |
| `-upstream-proxy` | *(empty)* | Route DevProxy's own egress through an upstream proxy: `http://`, `https://` or `socks5://host:port` |

When `-upstream-proxy` is not set, DevProxy falls back to the standard
`HTTPS_PROXY` / `HTTP_PROXY` / `ALL_PROXY` and `NO_PROXY` environment
variables. Upstream proxy credentials (`user:password@`) and PAC files are not
supported; traffic that would loop back into DevProxy's own listen address is
refused.

---

## 🧪 Comprehensive Testing Suite

DevProxy enforces multiple testing tiers in CI to guarantee zero-latency performance, memory safety, and protocol resilience:

```bash
# 1. Multi-OS Unit Tests with Data Race Detection
go test -v -race ./...

# 2. Performance & Zero-Allocation Microbenchmarks
go test -v -run='^$' -bench=. -benchmem ./pkg/ringbuffer ./pkg/analysis

# 3. Security Protocol Fuzz Testing
go test -v -fuzz=FuzzRingBuffer -fuzztime=10s ./pkg/ringbuffer
go test -v -fuzz=FuzzAhoCorasickMatcher -fuzztime=10s ./pkg/analysis
go test -v -fuzz=FuzzJWTInspection -fuzztime=10s ./pkg/analysis
go test -v -fuzz=FuzzLuhnValidation -fuzztime=10s ./pkg/analysis
go test -v -fuzz=FuzzLLMInspector -fuzztime=10s ./pkg/analysis

# 4. End-to-End (E2E) Proxy & Dashboard Integration Tests
go test -v -race ./test/e2e/...

# 5. Docker Packaging & Container Smoke Test
docker build -t devproxy:smoke .
```

---

## 🤖 Automated Code Quality & Security

DevProxy enforces strict static analysis, vulnerability detection, and performance benchmarking across every pull request:
- **[CodeRabbit AI](https://coderabbit.ai)**: Automated AI code reviewer providing contextual summaries, diff walkthroughs, and inline improvement suggestions.
- **[DeepSource](https://deepsource.com)**: Continuous static analysis detecting:
  - 🛡️ **Security Vulnerabilities**: Injection vectors, credential leaks, and insecure crypto usage.
  - ⚡ **Performance Bottlenecks**: Memory allocations, unbuffered I/O, and goroutine synchronization pitfalls.
  - 🔧 **Autofix**: Automated 1-click code fixes directly in PRs.
  - 📊 **Test Coverage Tracking**: Integrated coverage reporting via `.deepsource.toml` and GitHub Actions.
- **[Dependabot](https://github.com/dependabot)**: Automated weekly CVE vulnerability scanning and dependency updates for Go modules, Docker, and GitHub Actions.
- **[Gosec](https://github.com/securego/gosec)**: AST-based Go security audit reporting directly to GitHub's native Security tab.
- **[Release Drafter](https://github.com/release-drafter/release-drafter)**: Automatically categorizes merged PRs and drafts changelogs with contributor attribution.

---

## 👥 Contributors

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/Aditya-9-6"><img src="https://avatars.githubusercontent.com/u/105096058?v=4?s=100" width="100px;" alt="Aditya Dahale"/><br /><sub><b>Aditya Dahale</b></sub></a><br /><a href="https://github.com/Aditya-9-6/DevProxy/commits?author=Aditya-9-6" title="Code">💻</a> <a href="https://github.com/Aditya-9-6/DevProxy/commits?author=Aditya-9-6" title="Documentation">📖</a> <a href="#design-Aditya-9-6" title="Design">🎨</a> <a href="#maintenance-Aditya-9-6" title="Maintenance">🚧</a> <a href="#security-Aditya-9-6" title="Security">🛡️</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/varun-pahuja"><img src="https://avatars.githubusercontent.com/u/190826251?v=4?s=100" width="100px;" alt="Varun Pahuja"/><br /><sub><b>Varun Pahuja</b></sub></a><br /><a href="https://github.com/Aditya-9-6/DevProxy/commits?author=varun-pahuja" title="Code">💻</a> <a href="https://github.com/Aditya-9-6/DevProxy/commits?author=varun-pahuja" title="Documentation">📖</a> <a href="https://github.com/Aditya-9-6/DevProxy/commits?author=varun-pahuja" title="Tests">⚠️</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/akshat-lakhera"><img src="https://avatars.githubusercontent.com/u/211803516?v=4?s=100" width="100px;" alt="Akshat Lakhera"/><br /><sub><b>Akshat Lakhera</b></sub></a><br /><a href="https://github.com/Aditya-9-6/DevProxy/commits?author=akshat-lakhera" title="Code">💻</a> <a href="#design-akshat-lakhera" title="Design">🎨</a> <a href="https://github.com/Aditya-9-6/DevProxy/commits?author=akshat-lakhera" title="Documentation">📖</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind are welcome!

---

## 📜 License

MIT License. Designed for high-performance development environments.
