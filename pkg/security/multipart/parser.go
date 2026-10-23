package multipart

import (
	"io"
)

type Parser struct {
	matcher *BoundaryMatcher
	limits  Limits
}

func NewParser(boundary string, limits Limits) *Parser {
	return &Parser{
		matcher: NewBoundaryMatcher(boundary),
		limits:  limits,
	}
}

func (p *Parser) Parse(r io.Reader) error {
	return nil
}
