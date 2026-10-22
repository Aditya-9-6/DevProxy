package proxy

import (
	"fmt"
	"os"
	"runtime"
)

// RedirectionMode describes the network interception strategy.
type RedirectionMode string

const (
	ModeExplicitProxy RedirectionMode = "EXPLICIT_HTTP_PROXY"
	ModeEBPF          RedirectionMode = "EBPF_CGROUP"
	ModeIPTables      RedirectionMode = "IPTABLES_TRANSPARENT"
)

// EBPFManager coordinates transparent traffic redirection.
type EBPFManager struct {
	ProxyPort int
	PID       int
	CGroup    string
}

// NewEBPFManager initializes the eBPF / redirection coordinator.
func NewEBPFManager(proxyPort int) *EBPFManager {
	return &EBPFManager{
		ProxyPort: proxyPort,
		PID:       os.Getpid(),
		CGroup:    "/sys/fs/cgroup/unified",
	}
}

// GenerateEBPFInstructions returns the Linux eBPF cgroup program and iptables redirection script.
func (m *EBPFManager) GenerateEBPFInstructions() string {
	return fmt.Sprintf(`
# ==============================================================================
# DevProxy Transparent eBPF Redirection Blueprint (Linux / Docker)
# ==============================================================================
# To silently route traffic from specific Docker containers or local PIDs to
# DevProxy (port %d) without configuring HTTP_PROXY variables:

# Option A: eBPF sock_ops redirection (Zero kernel copy)
# Attach BPF_PROG_TYPE_SOCK_OPS to cgroup2:
# 1. Compile ebpf_redirect.c:
#    clang -O2 -target bpf -c ebpf_redirect.c -o ebpf_redirect.o
# 2. Attach to target container cgroup:
#    bpftool cgroup attach /sys/fs/cgroup/docker/<container_id> sock_ops pinned /sys/fs/bpf/devproxy_sockops

# Option B: Docker / Linux Container iptables redirection:
iptables -t nat -A OUTPUT -p tcp -m owner ! --uid-owner $(id -u) --dport 80 -j REDIRECT --to-port %d
iptables -t nat -A OUTPUT -p tcp -m owner ! --uid-owner $(id -u) --dport 443 -j REDIRECT --to-port %d

# Option C: Quick CLI environment variables:
export HTTP_PROXY="http://127.0.0.1:%d"
export HTTPS_PROXY="http://127.0.0.1:%d"
export ALL_PROXY="http://127.0.0.1:%d"
`, m.ProxyPort, m.ProxyPort, m.ProxyPort, m.ProxyPort, m.ProxyPort, m.ProxyPort)
}

// CheckEnvironmentSupport checks if the host OS supports eBPF socket redirection directly.
func (m *EBPFManager) CheckEnvironmentSupport() (bool, string) {
	if runtime.GOOS != "linux" {
		return false, fmt.Sprintf("eBPF socket filtering requires Linux kernel >= 5.4. Current OS: %s. Using HTTP_PROXY standard redirection.", runtime.GOOS)
	}
	if _, err := os.Stat("/sys/fs/bpf"); err != nil {
		return false, "/sys/fs/bpf filesystem is not mounted. Ensure bpffs is mounted."
	}
	return true, "Linux eBPF subsystem is available."
}
