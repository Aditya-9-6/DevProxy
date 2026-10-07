package proxy

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// ProxyServer defines the core proxy structure.
type ProxyServer struct {
	addr          string
	upstreamProxy *url.URL
	loopWarnOnce  sync.Once
}

// calculateJA3 computes the JA3 fingerprint from a ClientHelloInfo.
func calculateJA3(chi *tls.ClientHelloInfo) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(int(chi.SupportedVersions[0])))
	b.WriteString(",")

	var ciphers []string
	for _, c := range chi.CipherSuites {
		ciphers = append(ciphers, strconv.Itoa(int(c)))
	}
	b.WriteString(strings.Join(ciphers, "-"))
	b.WriteString(",")

	var exts []string
	for _, e := range chi.Extensions {
		exts = append(exts, strconv.Itoa(int(e)))
	}
	b.WriteString(strings.Join(exts, "-"))
	b.WriteString(",")

	var curves []string
	for _, c := range chi.SupportedCurves {
		curves = append(curves, strconv.Itoa(int(c)))
	}
	b.WriteString(strings.Join(curves, "-"))
	b.WriteString(",")

	var formats []string
	for _, f := range chi.SupportedPoints {
		formats = append(formats, strconv.Itoa(int(f)))
	}
	b.WriteString(strings.Join(formats, "-"))

	hash := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(hash[:])
}
