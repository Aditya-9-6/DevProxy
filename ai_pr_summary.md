## 🤖 Automated Solution for Issue #33

**Issue**: #33 - feat(proxy): Add gRPC & Protobuf Binary Stream Decoder in Dashboard

### 📝 Solution Overview
Implemented gRPC binary stream decoding by adding a dedicated gRPC parser in pkg/proxy/grpc.go, integrating it into the proxy's traffic capture flow, and updating the dashboard hub to handle gRPC-specific event metadata. The implementation uses sync.Pool for zero-allocation buffer management and provides a robust 5-byte header parsing mechanism for gRPC frames.

### 📦 Files Changed
- `pkg/proxy/grpc.go`
- `pkg/proxy/grpc_test.go`
- `pkg/dashboard/hub.go`

### 🛡️ Quality & Performance Invariants
- [x] Idiomatic Go concurrency and memory safety.
- [x] High throughput and streaming invariants preserved.
- [x] Comprehensive unit tests included.

---
*Closes #33*
