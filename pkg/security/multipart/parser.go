package multipart

import (
	"bufio"
	"bytes"
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
	reader := bufio.NewReader(r)
	partsCount := 0
	var headerBuf bytes.Buffer
	inHeader := true
	lineCount := 0

	for {
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if isPrefix {
			return ErrLineTooLong
		}

		lineCount++
		if len(line) > p.limits.MaxLineLength {
			return ErrLineTooLong
		}

		if p.matcher.MatchClose(line) || p.matcher.MatchFull(line) {
			if inHeader && headerBuf.Len() > 0 {
				_, err := ParseHeader(headerBuf.String(), p.limits.MaxHeaderBytes)
				if err != nil {
					return err
				}
				headerBuf.Reset()
			}
			partsCount++
			if partsCount > p.limits.MaxParts {
				return ErrPartLimitExceeded
			}
			inHeader = true
			continue
		}

		if inHeader {
			if len(line) == 0 {
				inHeader = false
				_, err := ParseHeader(headerBuf.String(), p.limits.MaxHeaderBytes)
				if err != nil {
					return err
				}
			} else {
				headerBuf.Write(line)
				headerBuf.WriteByte('\n')
				if headerBuf.Len() > p.limits.MaxHeaderBytes {
					return ErrHeaderTooLarge
				}
			}
		}
	}
	return nil
}
