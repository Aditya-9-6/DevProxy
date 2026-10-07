# Feature Proposal: JA3/JA4 Client TLS Fingerprinting

## 🎯 Objective
To enhance the passive security analysis capabilities of DevProxy, we should implement TLS Client Hello fingerprinting (JA3/JA4). This allows the proxy to detect the true nature of the client making requests, identifying malicious bots, scripts masking as browsers, or outdated vulnerable TLS libraries.

## 💡 The Fingerprinting Approach
When a client initiates a TLS connection, the `Client Hello` packet contains specific cipher suites, extensions, and elliptic curves in a deterministic order.

### 1. Passive Packet Inspection
- In `pkg/proxy/proxy.go`, immediately after accepting a TCP connection (before the TLS handshake completes), peek into the network buffer to read the raw `Client Hello` bytes.
- Parse the TLS record layer to extract:
  - TLS Version
  - Accepted Cipher Suites
  - Extension IDs
  - Supported Elliptic Curves & Formats

### 2. JA4 Hash Generation
- Implement the JA4 hashing standard.
- Concatenate the extracted values into the standard string format.
- Generate the MD5/SHA256 hashes to produce the final fingerprint string (e.g., `t13d1516h1_8daaf6152771_a04b12389146`).

### 3. Threat Intelligence Integration
- Attach the generated fingerprint to the `TrafficEvent` ringbuffer payload.
- Update the `SecurityRulesEngine` to check the fingerprint against a known blocklist of malicious actors, DDoS botnets, or outdated Python/Java libraries masquerading as modern web browsers.

## 🛠️ Implementation Steps
1. **Connection Peeking:** Create a `PeekConn` wrapper around `net.Conn` that allows reading the first ~2KB of bytes without consuming them from the stream (so the actual Go `crypto/tls` library can still process the handshake).
2. **TLS Parser:** Implement a lightweight, zero-allocation parser for the `Client Hello` bytes in `pkg/analysis/tls`.
3. **JA4 Algorithm:** Implement the specific JA4 formatting and hashing logic.
4. **UI Integration:** Display the detected Client OS/Browser and JA4 hash in the Web Dashboard's request details panel.

## ⚠️ Security & Constraints
- **Performance:** Peeking and parsing the byte stream must be highly optimized to maintain the zero-latency goals of the proxy. Avoid unnecessary heap allocations during the byte parsing phase.
