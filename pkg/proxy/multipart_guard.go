package proxy

import (
	"io"
	"github.com/Aditya-9-6/DevProxy/pkg/security/multipart"
)

type MultipartGuard struct {
	parser *multipart.Parser
}

func NewMultipartGuard(boundary string, limits multipart.Limits) *MultipartGuard {
	return &MultipartGuard{
		parser: multipart.NewParser(boundary, limits),
	}
}

func (g *MultipartGuard) Validate(r io.Reader) error {
	return g.parser.Parse(r)
}
