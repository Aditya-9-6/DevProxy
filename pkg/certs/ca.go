package certs

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func CalculateJA3(chi *tls.ClientHelloInfo) string {
	var versions []string
	versions = append(versions, strconv.Itoa(int(chi.SupportedVersions[0])))

	var ciphers []string
	for _, c := range chi.CipherSuites {
		ciphers = append(ciphers, strconv.Itoa(int(c)))
	}

	var extensions []string
	for _, e := range chi.SupportedVersions {
		extensions = append(extensions, strconv.Itoa(int(e)))
	}

	var curves []string
	for _, c := range chi.SupportedCurves {
		curves = append(curves, strconv.Itoa(int(c)))
	}

	var formats []string
	for _, f := range chi.SupportedPoints {
		formats = append(formats, strconv.Itoa(int(f)))
	}

	ja3 := fmt.Sprintf("%s,%s,%s,%s,%s", strings.Join(versions, "-"), strings.Join(ciphers, "-"), strings.Join(extensions, "-"), strings.Join(curves, "-"), strings.Join(formats, "-"))
	hash := md5.Sum([]byte(ja3))
	return hex.EncodeToString(hash[:])
}
