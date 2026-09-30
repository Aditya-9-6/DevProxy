# ⚡ DevProxy v1.0.0: Zero-Latency Security Proxy & AI/LLM Traffic Inspector

We are excited to announce the first production release of **DevProxy**! 

Traditional debugging and security scanning proxies insert synchronous inspection into the local request/response loop, adding hundreds of milliseconds of latency and breaking developer workflows. 

**DevProxy** resolves this by completely **decoupling line-rate packet forwarding from vulnerability analysis** using a lock-free circular ring buffer and asynchronous worker pools.

---

### 🌟 Key Highlights in v1.0.0

- ⚡ **Zero Added Latency:** Full-duplex streaming proxy engine with immediate line-rate packet forwarding (< 1ms network hop overhead).
- 🤖 **AI & LLM Traffic Inspector:** Real-time token tracking, 2026 model pricing cost estimation (USD), runaway token alerting (>8,000 tokens), and prompt data leak detection (Credit Cards, SSNs, AWS keys) across OpenAI, Anthropic, Gemini, Groq, Mistral, and local Ollama.
- 🔄 **Request Replay & Semantic Diff:** Re-execute any past HTTP request with parameter overrides and inspect real-time header and body diffs against original upstream responses.
- 💥 **Mocking & Chaos Engineering:** Map Local (serve mock JSON), Map Remote (reverse proxy redirection), fault injection (latency, error rates), and real-world network throttling profiles (Slow 3G, Fast 3G, LTE, Offline).
- 📜 **OpenAPI Contract Validator:** Passively compares live traffic against OpenAPI 3.0 / Swagger specifications to detect Shadow APIs, undocumented endpoints, and schema drift.
- 🛡️ **Passive Security Rules Engine:** Bitwise Radix Trie CIDR blocklist, Aho-Corasick multi-pattern secret scanner, JWT inspector, PII defense (Luhn mod-10 verified credit cards), and cookie security auditor (HttpOnly, SameSite, Secure).
- 🎨 **Reactive Dark-Mode Dashboard:** Single-binary zero-dependency web interface with live WebSocket packet streaming and instant cURL export.

---

### 📦 Quick Start & Installation

#### Option 1: Go Install
```
go install github.com/Aditya-9-6/DevProxy/cmd/devproxy@latest
devproxy -port 8080 -web-port 8081
```

#### Option 2: Pre-compiled Binaries
Download the pre-compiled binary for your architecture from the **Assets** section below:
- **Windows (x64 / ARM64):** `devproxy-windows-amd64.exe` / `devproxy-windows-arm64.exe`
- **macOS (Apple Silicon & Intel):** `devproxy-darwin-arm64` / `devproxy-darwin-amd64`
- **Linux (x64 / ARM64):** `devproxy-linux-amd64` / `devproxy-linux-arm64`

#### Option 3: Docker
```
docker-compose up
```

---

### 🎃 Hacktoberfest Ready!
DevProxy is an open-source Hacktoberfest project. Check out our open issues (https://github.com/Aditya-9-6/DevProxy/issues) to contribute!
