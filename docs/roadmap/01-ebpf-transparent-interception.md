# Feature Proposal: eBPF Transparent Interception (Zero-Configuration)

## 🎯 Objective
Currently, DevProxy requires developers to manually configure `HTTP_PROXY`/`HTTPS_PROXY` environment variables and manually install/trust the `devproxy-ca.crt` Root CA on their machines. We aim to achieve "Zero-Configuration" interception using Linux eBPF, putting DevProxy on par with enterprise observability tools like Pixie and Cilium.

## 💡 The eBPF Approach
Instead of acting as a traditional MITM proxy at the network layer, DevProxy will attach directly to the Linux Kernel and userspace shared libraries to intercept data *before* it is encrypted and *after* it is decrypted.

### 1. `sock_ops` & `sk_msg` BPF Programs (Container Routing)
- Use `cilium/ebpf` to attach to `cgroup` socket operations.
- Automatically detect new TCP connections originating from local Docker containers (e.g., matching a specific namespace or cgroup).
- Silently redirect those sockets to the DevProxy engine's listening port at the kernel level. This eliminates the need for iptables rules or `HTTP_PROXY` variables inside containers.

### 2. Uprobe TLS Interception (No Root CA Required)
- Use eBPF Uprobes (User-space Probes) to hook into the `SSL_read` and `SSL_write` functions of OpenSSL, BoringSSL (used by Node.js/Python), and the `crypto/tls` package in Go.
- DevProxy will read the raw, unencrypted byte arrays directly from the application's memory space just before the TLS library encrypts them to send over the network.
- **Benefit:** Developers never need to install the Root CA. The application talks to the real upstream server with the real TLS certificate, but DevProxy gets a perfect clone of the plaintext data for the RingBuffer security analysis.

## 🛠️ Implementation Steps
1. **Scaffold `pkg/ebpf`:** Add the `cilium/ebpf` dependency.
2. **C-code BPF Programs:** Write the eBPF C programs (`bpf_bpfel.o`) for `uprobe/SSL_write` and `uprobe/SSL_read`. Use `bpf2go` to compile them into Go stubs.
3. **Ring Buffer Integration:** Create an eBPF Perf Event array that streams intercepted byte payloads from the kernel directly into DevProxy's existing `ringbuffer.RingBuffer`.
4. **PID Tracking:** Map the intercepted packets to specific PIDs and Container IDs to enrich the Web UI dashboard with process-level attribution.

## ⚠️ Security & Constraints
- **Privilege:** eBPF requires DevProxy to be executed as `root` (or with `CAP_SYS_ADMIN` / `CAP_BPF`).
- **OS Dependency:** This feature is strictly Linux-only (Kernel 5.8+ recommended). Windows and macOS users will continue to use the standard MITM proxy fallback.
