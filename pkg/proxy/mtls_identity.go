package proxy

import (
	"crypto/tls"
	"errors"
	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

func ExtractSPIFFEID(state tls.ConnectionState) (spiffe.ID, error) {
	if len(state.PeerCertificates) == 0 {
		return spiffe.ID{}, errors.New("no peer certificates found")
	}
	cert := state.PeerCertificates[0]
	if len(cert.URIs) == 0 {
		return spiffe.ID{}, errors.New("no URIs found in peer certificate")
	}
	for _, uri := range cert.URIs {
		id, err := spiffe.ParseID(uri.String())
		if err == nil {
			return id, nil
		}
	}
	return spiffe.ID{}, errors.New("no valid SPIFFE ID found in URIs")
}
