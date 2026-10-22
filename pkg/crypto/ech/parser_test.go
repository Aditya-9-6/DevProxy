package ech

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func buildClientHello(extensions []byte) []byte {
	ch := make([]byte, 38)
	ch[0] = 1 // Handshake Type: ClientHello

	// Session ID (1 byte length + data)
	ch = append(ch, 0)

	// Cipher Suites (2 bytes length + data)
	ch = append(ch, []byte{0, 2, 0x13, 0x01}...) // length 2, TLS_AES_128_GCM_SHA256

	// Compression Methods (1 byte length + data)
	ch = append(ch, []byte{1, 0}...) // length 1, null compression

	// Extensions length
	extsLen := make([]byte, 2)
	binary.BigEndian.PutUint16(extsLen, uint16(len(extensions)))
	ch = append(ch, extsLen...)

	// Extensions
	ch = append(ch, extensions...)

	return ch
}

func TestParseECHExtension(t *testing.T) {
	ext1 := make([]byte, 4)
	binary.BigEndian.PutUint16(ext1, 0x0000) // Server Name
	binary.BigEndian.PutUint16(ext1[2:], 0)  // length 0

	ext2 := make([]byte, 4)
	binary.BigEndian.PutUint16(ext2, 0xfe0d) // ECH
	binary.BigEndian.PutUint16(ext2[2:], 5)  // length 5
	ext2 = append(ext2, []byte{1, 2, 3, 4, 5}...)

	validECHExt := append(ext1, ext2...)
	validCH := buildClientHello(validECHExt)

	noECHExt := ext1
	noECHCH := buildClientHello(noECHExt)

	tests := []struct {
		name        string
		clientHello []byte
		want        []byte
		wantErr     error
	}{
		{
			name:        "Valid ECH Extension",
			clientHello: validCH,
			want:        []byte{1, 2, 3, 4, 5},
			wantErr:     nil,
		},
		{
			name:        "No ECH Extension",
			clientHello: noECHCH,
			want:        nil,
			wantErr:     ErrECHNotFound,
		},
		{
			name:        "Client Hello Too Short",
			clientHello: []byte{1, 0, 0},
			want:        nil,
			wantErr:     errors.New("client hello too short"),
		},
		{
			name: "Malformed Cipher Suites",
			clientHello: func() []byte {
				ch := make([]byte, 38)
				ch = append(ch, 0) // sid
				ch = append(ch, []byte{0}...) // malformed cipher suites (length 0, but needs at least 2 bytes length prefix)
				return ch
			}(),
			want:    nil,
			wantErr: errors.New("malformed cipher suites"),
		},
		{
			name: "Malformed Compression Methods",
			clientHello: func() []byte {
				ch := make([]byte, 38)
				ch = append(ch, 0) // sid
				ch = append(ch, []byte{0, 2, 0, 0}...) // cipher suites
				// missing compression methods
				return ch
			}(),
			want:    nil,
			wantErr: errors.New("malformed compression methods"),
		},
        {
			name: "Malformed Extension Length",
			clientHello: func() []byte {
				ch := make([]byte, 38)
				ch = append(ch, 0) // sid
				ch = append(ch, []byte{0, 2, 0, 0}...) // cipher suites
                ch = append(ch, []byte{1, 0}...) // compression methods

                exts := []byte{0, 4, 0, 0, 0, 10} // length 4, but missing 10 bytes
                ch = append(ch, exts...)
				return ch
			}(),
			want:    nil,
			wantErr: errors.New("malformed extension length"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseECHExtension(tt.clientHello)
			if err != tt.wantErr && err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				if err != tt.wantErr {
					t.Errorf("ParseECHExtension() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			} else if (err == nil && tt.wantErr != nil) || (err != nil && tt.wantErr == nil) {
				t.Errorf("ParseECHExtension() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("ParseECHExtension() got = %v, want %v", got, tt.want)
			}
		})
	}
}
