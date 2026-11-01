package frame

import (
	"io"
)

type Decoder struct {
	rdr io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{rdr: r}
}