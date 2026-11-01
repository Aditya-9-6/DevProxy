package certverify

import (
	"bytes"
	"github.com/Aditya-9-6/DevProxy/pkg/tls/certcomp"
)

// ChainVerifier parses, verifies, and bounds-checks incoming certificate chains.
type ChainVerifier struct {
	Limits certcomp.Limits
}

func NewChainVerifier(limits certcomp.Limits) *ChainVerifier {
	return &ChainVerifier{Limits: limits}
}

// VerifyCompressedChain decompresses and validates certificate vector counts and bounds.
func (v *ChainVerifier) VerifyCompressedChain(msg *certcomp.CompressedCertificateMessage) ([]byte, error) {
	codec, ok := certcomp.GetCodec(msg.Algorithm)
	if !ok {
		return nil, certcomp.ErrUnsupportedAlgorithm
	}

	out := new(bytes.Buffer)
	_, err := codec.Decompress(out, msg.CompressedCertData, v.Limits.MaxUncompressedSize)
	if err != nil {
		return nil, err
	}

	if uint32(out.Len()) != msg.UncompressedLength {
		return nil, certcomp.ErrMalformedMessage
	}

	return out.Bytes(), nil
}