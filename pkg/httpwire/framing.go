package httpwire

import (
	"bufio"
	"bytes"
	"errors"
	"strconv"
)

var (
	ErrInvalidHeader = errors.New("httpwire: invalid header format")
	ErrContentLength = errors.New("httpwire: invalid or missing content length")
	ErrChunkedDecode = errors.New("httpwire: invalid chunked encoding")
)

// Frame represents a structured HTTP/1.1 or generic framing view.
type Frame struct {
	Method      string
	URI         string
	Proto       string
	StatusCode  int
	Status      string
	Headers     [][2]string
	Body        []byte
	IsRequest   bool
	Chunked     bool
}

// ParseRequestFraming parses a raw HTTP/1.1 request into a Frame structure with zero/minimal allocations.
func ParseRequestFraming(r *bufio.Reader) (*Frame, error) {
	line, isPrefix, err := r.ReadLine()
	if err != nil {
		return nil, err
	}
	if isPrefix {
		return nil, ErrInvalidHeader
	}

	parts := bytes.SplitN(line, []byte{' '}, 3)
	if len(parts) < 3 {
		return nil, ErrInvalidHeader
	}

	frame := &Frame{
		Method:    string(parts[0]),
		URI:       string(parts[1]),
		Proto:     string(parts[2]),
		IsRequest: true,
	}

	err = parseHeaders(r, frame)
	if err != nil {
		return nil, err
	}

	return frame,
		nil
}

// ParseResponseFraming parses a raw HTTP/1.1 response.
func ParseResponseFraming(r *bufio.Reader) (*Frame, error) {
	line, isPrefix, err := r.ReadLine()
	if err != nil {
		return nil, err
	}
	if isPrefix {
		return nil, ErrInvalidHeader
	}

	parts := bytes.SplitN(line, []byte{' '}, 3)
	if len(parts) < 2 {
		return nil, ErrInvalidHeader
	}

	statusCode, err := strconv.Atoi(string(parts[1]))
	if err != nil {
		return nil, ErrInvalidHeader
	}

	proto := string(parts[0])
	status := ""
	if len(parts) > 2 {
		status = string(parts[2])
	}

	frame := &Frame{
		Proto:      proto,
		StatusCode: statusCode,
		Status:     status,
		IsRequest:  false,
	}

	err = parseHeaders(r, frame)
	if err != nil {
		return nil, err
	}

	return frame, nil
}

func parseHeaders(r *bufio.Reader, frame *Frame) error {
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil {
			return err
		}
		if isPrefix {
			return ErrInvalidHeader
		}

		if len(line) == 0 {
			break
		}

		parts := bytes.SplitN(line, []byte{':'}, 2)
		if len(parts) != 2 {
			return ErrInvalidHeader
		}

		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		frame.Headers = append(frame.Headers, [2]string{key, val})

		if bytes.EqualFold(parts[0], []byte("Transfer-Encoding")) && bytes.Contains(bytes.ToLower(parts[1]), []byte("chunked")) {
			frame.Chunked = true
		}
	}
	return nil
}
