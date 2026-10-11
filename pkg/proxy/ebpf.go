package proxy

import (
	"fmt"
	"sync"
)

// EBPFManager handles the lifecycle of eBPF programs for traffic interception.
type EBPFManager struct {
	proxyPort int
	mu        sync.RWMutex
}

// NewEBPFManager creates a new manager for the given proxy port.
func NewEBPFManager(proxyPort int) *EBPFManager {
	return &EBPFManager{proxyPort: proxyPort}
}

// GenerateEBPFInstructions returns the setup guide for transparent redirection.
func (m *EBPFManager) GenerateEBPFInstructions() string {
	return fmt.Sprintf(`
================================================================================
 eBPF TC INGRESS REDIRECTION (TRANSPARENT PROXYING)
================================================================================
To enable transparent L4 interception, attach the DevProxy TC filter:

1. Compile the eBPF program:
   clang -O2 -g -target bpf -c tc_ingress.c -o tc_ingress.o

2. Attach to your primary network interface (e.g., eth0):
   sudo tc qdisc add dev eth0 clsact
   sudo tc filter add dev eth0 ingress bpf da obj tc_ingress.o sec tc_ingress

3. DevProxy will now receive mirrored packets on port %d via the eBPF map.
================================================================================`, m.proxyPort)
}
