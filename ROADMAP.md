# 🚀 DevProxy Enterprise Roadmap

DevProxy is continually evolving to meet the demands of industry-standard enterprise security and observability. The following advanced features have detailed implementation proposals drafted in the `docs/roadmap/` directory.

We welcome contributions and discussion on these issues!

## 📌 Proposed Features

### 1. Zero-Configuration eBPF Transparent Interception
*   **Status:** Proposed / Needs Help
*   **Details:** [docs/roadmap/01-ebpf-transparent-interception.md](docs/roadmap/01-ebpf-transparent-interception.md)
*   **Goal:** Use Linux eBPF uprobes and socket filters to automatically intercept unencrypted traffic from Docker containers and CLI tools without requiring manual Root CA installation or `HTTPS_PROXY` environment variables.

### 2. Local Machine Learning Behavioral Anomaly Detection
*   **Status:** Proposed / Architecture Drafted
*   **Details:** [docs/roadmap/02-ml-behavioral-anomaly-detection.md](docs/roadmap/02-ml-behavioral-anomaly-detection.md)
*   **Goal:** Embed the ONNX runtime to execute lightweight Isolation Forests or Autoencoders against the RingBuffer stream, identifying undocumented Shadow APIs and BOLA attacks by detecting deviations from normal traffic baselines.

### 3. gRPC and Protobuf Dynamic Reflection Decoder
*   **Status:** Proposed
*   **Details:** [docs/roadmap/03-grpc-protobuf-decoder.md](docs/roadmap/03-grpc-protobuf-decoder.md)
*   **Goal:** Decode raw HTTP/2 gRPC streams into readable JSON in the Web UI by dynamically querying the upstream service's gRPC Server Reflection endpoint, allowing security rules to run against typed Protobuf fields.

### 4. JA3/JA4 Client TLS Fingerprinting
*   **Status:** Proposed
*   **Details:** [docs/roadmap/04-tls-fingerprinting.md](docs/roadmap/04-tls-fingerprinting.md)
*   **Goal:** Passively parse TLS Client Hello packets to generate JA4 fingerprints, enabling the detection of malicious bots or vulnerable clients attempting to masquerade as standard web browsers.

---

*Interested in tackling one of these? Check out the individual markdown files for implementation steps and constraints!*
