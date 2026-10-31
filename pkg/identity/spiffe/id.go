package spiffe

import (
	"errors"
	"strings"
)

type ID struct {
	TrustDomain string
	Path        string
}

func ParseID(raw string) (ID, error) {
	if !strings.HasPrefix(raw, "spiffe://") {
		return ID{}, errors.New("invalid spiffe id scheme")
	}
	parts := strings.SplitN(strings.TrimPrefix(raw, "spiffe://"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return ID{}, errors.New("empty trust domain")
	}
	path := ""
	if len(parts) > 1 {
		path = "/" + parts[1]
	}
	return ID{
		TrustDomain: parts[0],
		Path:        path,
	}, nil
}

func (id ID) String() string {
	return "spiffe://" + id.TrustDomain + id.Path
}
