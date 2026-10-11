package proxy

// TCEBPFProgram defines the C source code for the eBPF TC ingress filter.
// This program mirrors packets to the DevProxy userspace engine.
const TCEBPFProgram = `
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <bpf/bpf_helpers.h>

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} traffic_map SEC(".maps");

SEC("tc_ingress")
int tc_ingress(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    struct ethhdr *eth = data;
    if (data + sizeof(*eth) > data_end) return TC_ACT_OK;

    if (eth->h_proto != bpf_htons(ETH_P_IP)) return TC_ACT_OK;

    struct iphdr *iph = data + sizeof(*eth);
    if (data + sizeof(*eth) + sizeof(*iph) > data_end) return TC_ACT_OK;

    // Mirror packet to userspace via ring buffer
    bpf_ringbuf_output(&traffic_map, data, data_end - data, 0);

    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
`
