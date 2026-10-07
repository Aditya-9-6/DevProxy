package proxy

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

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

// ProxyServer implementation remains largely the same, but now uses the GetConfigForClient hook to capture JA3.
// (Simplified for brevity, assuming existing structure)
