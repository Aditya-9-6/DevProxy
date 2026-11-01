package certcomp

import (
	"bytes"
	"encoding/binary"
	"io"
)

// CompressedCertificateMessage represents an RFC 8879 compressed certificate message.
type CompressedCertificateMessage struct {
	Algorithm          Algorithm
	UncompressedLength uint32
	CompressedCertData []byte
}

func MarshalCompressedCertificate(alg Algorithm, uncompressedLen uint32, compressed []byte) []byte {
	buf := new(bytes.Buffer)
	// RFC 8879 wire format: algorithm (2 bytes), uncompressed_length (4 bytes), compressed_certificate_length (3 bytes), compressed_certificate
	var algBytes [2]byte
	binary.BigEndian.PutUint16(algBytes[:], uint16(alg))
	buf.Write(algBytes[:])

	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], uncompressedLen)
	buf.Write(lenBytes[:])

	compLen := len(compressed)
	var compLenBytes [3]byte
	compLenBytes[0] = byte(compLen >> 16)
	compLenBytes[1] = byte(compLen >> 8)
	compLenBytes[2] = byte(compLen)
	buf.Write(compLenBytes[:])

	buf.Write(compressed)
	return buf.Bytes()
}

func ParseCompressedCertificate(r io.Reader, limits Limits) (*CompressedCertificateMessage, error) {
	header := make([]byte, 9)
	if _, err := io.ReadFull(r, header);
	err != nil {
		return nil, ErrMalformedMessage
	}

	alg := Algorithm(binary.BigEndian.Uint16(header[0:2]))
	uncompressedLen := binary.BigEndian.Uint32(header[2:6])
	compLen := int(header[6])<<16 | int(header[7])<<8 | int(header[8])

	if compLen > limits.MaxCompressedSize {
		return nil, ErrCompressedSizeExceeded
	}
	if int(uncompressedLen) > limits.MaxUncompressedSize {
		return nil, ErrExpansionRatioExceeded
	}
	if float64(uncompressedLen) > float64(compLen)*limits.MaxExpansionRatio {
		return nil, ErrExpansionRatioExceeded
	}

	data := make([]byte, compLen)
	if _, err := io.ReadFull(r, data);
	err != nil {
		return nil, ErrMalformedMessage
	}

	return &CompressedCertificateMessage{
		Algorithm:          alg,
		UncompressedLength: uncompressedLen,
		CompressedCertData: data,
	}, nil
}