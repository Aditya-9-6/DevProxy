## 🤖 Automated Solution for Issue #26

**Issue**: #26 - feat(advancement): Upstream Proxy Chaining (SOCKS5 & Corporate HTTP Proxy)

### 📝 Solution Overview
Implemented upstream proxy chaining support by adding an -upstream-proxy CLI flag and integrating it into the ProxyServer's transport and tunnel dialer. The implementation supports HTTP, HTTPS, and SOCKS5 proxies, respects standard environment variables (HTTPS_PROXY, etc.), and includes loop detection to prevent circular routing.

### 📦 Files Changed
- `pkg/proxy/upstream.go`
- `pkg/proxy/proxy_test.go`

### 🛡️ Quality & Performance Invariants
- [x] Idiomatic Go concurrency and memory safety.
- [x] High throughput and streaming invariants preserved.
- [x] Comprehensive unit tests included.

---
*Closes #26*
