## 🤖 Automated Solution for Issue #453

**Issue**: #453 - feat(security): Implement dual-interpretation HTTP/1.1 request-smuggling guard

### 📝 Solution Overview
Implemented the dual-interpretation HTTP/1.1 request-smuggling guard and supporting HTTP framing verification modules under devproxy_agent/security/httpframing and devproxy_agent/proxy/http1, ensuring zero-allocation streaming patterns and strict compliance with CL.TE, TE.CL, and boundary limits. Added comprehensive table-driven unit tests verifying parser correctness, verdict enforcement, and adverse request smuggling scenarios.

### 📦 Files Changed
- `devproxy_agent/security/httpframing/__init__.py`
- `devproxy_agent/security/httpframing/verdict.py`
- `devproxy_agent/security/httpframing/parser.py`
- `devproxy_agent/security/httpframing/parser_test.py`
- `devproxy_agent/proxy/http1/__init__.py`
- `devproxy_agent/proxy/http1/request_guard.py`
- `devproxy_agent/proxy/http1/request_guard_test.py`

### 🛡️ Quality & Performance Invariants
- [x] Idiomatic Python concurrency and memory safety.
- [x] High throughput and streaming invariants preserved.
- [x] Comprehensive unit tests included.

---
*Closes #453*
