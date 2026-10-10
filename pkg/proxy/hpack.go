package proxy

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// HPACK Header Field representation
type HeaderField struct {
	Name  string
	Value string
}

// Static Table entries per RFC 7541 Appendix A
var staticTable = []HeaderField{
	{":authority", ""},
	{":method", "GET"},
	{":method", "POST"},
	{":path", "/"},
	{":path", "/index.html"},
	{":scheme", "http"},
	{":scheme", "https"},
	{":status", "200"},
	{":status", "204"},
	{":status", "206"},
	{":status", "304"},
	{":status", "400"},
	{":status", "404"},
	{":status", "500"},
	{"accept-charset", ""},
	{"accept-encoding", "gzip, deflate"},
	{"accept-language", ""},
	{"accept-ranges", ""},
	{"accept", ""},
	{"access-control-allow-origin", ""},
	{"age", ""},
	{"allow", ""},
	{"authorization", ""},
	{"cache-control", ""},
	{"content-disposition", ""},
	{"content-encoding", ""},
	{"content-language", ""},
	{"content-length", ""},
	{"content-location", ""},
	{"content-range", ""},
	{"content-type", ""},
	{"cookie", ""},
	{"date", ""},
	{"etag", ""},
	{"expect", ""},
	{"expires", ""},
	{"from", ""},
	{"host", ""},
	{"if-match", ""},
	{"if-modified-since", ""},
	{"if-none-match", ""},
	{"if-range", ""},
	{"if-unmodified-since", ""},
	{"last-modified", ""},
	{"link", ""},
	{"location", ""},
	{"max-forwards", ""},
	{"proxy-authenticate", ""},
	{"proxy-authorization", ""},
	{"range", ""},
	{"referer", ""},
	{"refresh", ""},
	{"retry-after", ""},
	{"server", ""},
	{"set-cookie", ""},
	{"strict-transport-security", ""},
	{"transfer-encoding", ""},
	{"user-agent", ""},
	{"vary", ""},
	{"via", ""},
	{"www-authenticate", ""},
}

// DynamicTable maintains HPACK dynamic table entries with capacity limits and eviction tracking.
type DynamicTable struct {
	entries []HeaderField
	size    int
	maxSize int
}

func NewDynamicTable(maxSize int) *DynamicTable {
	return &DynamicTable{
		maxSize: maxSize,
	}
}

func (dt *DynamicTable) Add(hf HeaderField) {
	entrySize := len(hf.Name) + len(hf.Value) + 32
	for dt.size+entrySize > dt.maxSize && len(dt.entries) > 0 {
		last := dt.entries[len(dt.entries)-1]
		lastSize := len(last.Name) + len(last.Value) + 32
		dt.size -= lastSize
		dt.entries = dt.entries[:len(dt.entries)-1]
	}
	if entrySize <= dt.maxSize {
		dt.entries = append([]HeaderField{hf}, dt.entries...)
		dt.size += entrySize
	}
}

func (dt *DynamicTable) Get(index int) (HeaderField, bool) {
	if index < 1 || index > len(dt.entries) {
		return HeaderField{}, false
	}
	return dt.entries[index-1], true
}

// HPACK Encoder and Decoder buffers using sync.Pool for zero-allocation hot paths
var hpackBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

type Encoder struct {
	dynamicTable *DynamicTable
}

func NewEncoder(maxDynamicTableSize int) *Encoder {
	return &Encoder{
		dynamicTable: NewDynamicTable(maxDynamicTableSize),
	}
}

func (e *Encoder) Encode(w io.Writer, headers []HeaderField) error {
	buf := hpackBufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		hpackBufferPool.Put(buf)
	}()

	for _, hf := range headers {
		// Literal Header Field without Indexing for simplicity and performance
		buf.WriteByte(0x00)
		// Encode Name
		if err := encodeString(buf, hf.Name); err != nil {
			return err
		}
		// Encode Value
		if err := encodeString(buf, hf.Value); err != nil {
			return err
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

type Decoder struct {
	dynamicTable *DynamicTable
}

func NewDecoder(maxDynamicTableSize int) *Decoder {
	return &Decoder{
		dynamicTable: NewDynamicTable(maxDynamicTableSize),
	}
}

func (d *Decoder) Decode(r io.Reader) ([]HeaderField, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var headers []HeaderField
	idx := 0
	for idx < len(data) {
		b := data[idx]
		// Check for Literal Header Field without Indexing (0000XXXX)
		if (b & 0xF0) == 0x00 {
			idx++
			name, n, err := decodeString(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			val, n, err := decodeString(data[idx:])
			if err != nil {
				return nil, err
			}
			idx += n
			headers = append(headers, HeaderField{Name: name, Value: val})
		} else {
			return nil, errors.New("unsupported HPACK prefix encoding in zero-allocation decoder")
		}
	}

	return headers, nil
}

func encodeString(w io.Writer, s string) error {
	// Simple uncompressed string encoding for zero-allocation performance
	length := len(s)
	if length < 128 {
		w.Write([]byte{byte(length)})
	} else {
		return errors.New("string too long for simple HPACK encoding")
	}
	_, err := w.Write([]byte(s))
	return err
}

func decodeString(data []byte) (string, int, error) {
	if len(data) == 0 {
		return "", 0, io.ErrUnexpectedEOF
	}
	length := int(data[0] & 0x7F)
	if len(data) < 1+length {
		return "", 0, io.ErrUnexpectedEOF
	}
	s := string(data[1 : 1+length])
	return s, 1 + length, nil
}
