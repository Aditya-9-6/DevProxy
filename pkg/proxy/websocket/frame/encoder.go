package frame

import (
	"io"
)

type Encoder struct {
	wtr io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{wtr: w}
}