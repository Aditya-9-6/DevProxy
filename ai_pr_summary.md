## 🤖 Automated Solution for Issue #450

**Issue**: #450 - feat(security): Implement HTTP/1.1 framing ambiguity firewall against request smuggling

### 📝 Solution Overview
Implemented the Python equivalent of the HTTP/1.1 framing ambiguity firewall against request smuggling (Issue #450) including strict validation of Content-Length, Transfer-Encoding, conflicting framing headers, and chunked transfer encodings, with robust unit tests.

### 📦 Files Changed
- `devproxy_agent/security/__init__.py`
- `devproxy_agent/security/http1framing.py`
- `tests/test_http1framing.py`

### 🛡️ Quality & Performance Invariants
- [x] Idiomatic Python concurrency and memory safety.
- [x] High throughput and streaming invariants preserved.
- [x] Comprehensive unit tests included.

---
*Closes #450*
