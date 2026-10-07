# Feature Proposal: gRPC and Protobuf Dynamic Reflection Decoder

## 🎯 Objective
DevProxy currently supports native HTTP/2 routing, meaning gRPC streams pass through the proxy correctly. However, in the Web UI dashboard, gRPC payloads are displayed as raw binary/gibberish. We need to implement dynamic Protobuf decoding so developers and security rules can inspect gRPC traffic as readable JSON.

## 💡 The gRPC Decoder Approach
gRPC uses Protocol Buffers (Protobuf) encoded over HTTP/2 framing. To make this readable without requiring developers to manually provide `.proto` schemas for every service, we will leverage gRPC Server Reflection.

### 1. HTTP/2 Frame Interception
- Extend `pkg/proxy` to parse HTTP/2 `DATA` frames.
- Reassemble fragmented gRPC messages by reading the standard 5-byte gRPC header (1 byte compression flag + 4 byte length).

### 2. Dynamic Protobuf Reflection
- Implement a client in `pkg/analysis/grpc` that communicates with the upstream service's gRPC Server Reflection Extension (`grpc.reflection.v1alpha.ServerReflection`).
- Automatically query the reflection service to retrieve the `FileDescriptorProto` for the intercepted method.

### 3. Dynamic JSON Transcoding
- Use `google.golang.org/protobuf/types/dynamicpb` to dynamically unmarshal the raw binary payload using the retrieved descriptors.
- Convert the dynamic message into human-readable JSON using `protojson.Marshal`.
- Store both the raw binary (for exact upstream forwarding) and the transcoded JSON (for the RingBuffer and Web UI) in the `TrafficEvent`.

## 🛠️ Implementation Steps
1. **gRPC Frame Parser:** Implement the 5-byte header parser in `pkg/proxy/proxy.go`.
2. **Reflection Client:** Create a reflection querying client using `github.com/jhump/protoreflect`.
3. **Caching Layer:** Cache retrieved `FileDescriptorSet` objects in memory so DevProxy only queries the reflection endpoint once per service method.
4. **Analysis Pipeline Update:** Update the `SecurityRulesEngine` to run its Aho-Corasick and WASM rules against the transcoded JSON string rather than the compressed binary.

## ⚠️ Security & Constraints
- **Fallback:** If the upstream service has Server Reflection disabled, DevProxy must gracefully fallback to displaying raw binary or attempting a "best-effort" raw protobuf decode (extracting field numbers and wire types without names).
