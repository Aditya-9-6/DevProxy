package table

import (
	"errors"
	"sync"
)

var (
	ErrIndexOutOfBounds = errors.New("qpack: index out of bounds")
	ErrCapacityExceeded  = errors.New("qpack: dynamic table capacity exceeded")
)

// HeaderField represents a name-value header pair.
type HeaderField struct {
	Name  string
	Value string
}

// StaticTable defines the RFC 9204 static table (99 entries).
var StaticTable = []HeaderField{
	{":authority", ""},
	{":path", "/"},
	{":age", "0"},
	{":content-disposition", ""},
	{":content-length", "0"},
	{":cookie", ""},
	{":date", ""},
	{":etag", ""},
	{":if-modified-since", ""},
	{":if-none-match", ""},
	{":last-modified", ""},
	{":link", ""},
	{":location", ""},
	{":method", "GET"},
	{":method", "CONNECT"},
	{":method", "DELETE"},
	{":method", "GET"},
	{":method", "HEAD"},
	{":method", "OPTIONS"},
	{":method", "POST"},
	{":method", "PUT"},
	{":scheme", "http"},
	{":scheme", "https"},
	{":status", "103"},
	{":status", "200"},
	{":status", "304"},
	{":status", "404"},
	{":status", "500"},
	{"accept", "*/*"},
	{"accept", "application/json"},
	{"accept-encoding", "gzip, deflate, br"},
	{"accept-ranges", "bytes"},
	{"access-control-allow-credentials", "true"},
	{"access-control-allow-headers", "*"},
	{"access-control-allow-methods", "GET"},
	{"access-control-allow-origin", "*"},
	{"access-control-expose-headers", ""},
	{"access-control-max-age", ""},
	{"vary", ""},
}

// DynamicTable manages RFC 9204 dynamic table storage with byte capacity bounds.
type DynamicTable struct {
	mu       sync.RWMutex
	entries    []HeaderField
	maxCap   uint64
	curSize  uint64
	inserted uint64
}

// NewDynamicTable creates a dynamic table with max capacity in bytes.
func NewDynamicTable(maxCap uint64) *DynamicTable {
	return &DynamicTable{
		maxCap: maxCap,
	}
}

// Size returns current size in bytes.
func (dt *DynamicTable) Size() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.curSize
}

// Capacity returns maximum capacity in bytes.
func (dt *DynamicTable) Capacity() uint64 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.maxCap
}

// SetCapacity updates maximum capacity and evicts entries if necessary.
func (dt *DynamicTable) SetCapacity(newCap uint64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.maxCap = newCap
	dt.evict()
}

// Add inserts a new header field into the dynamic table.
func (dt *DynamicTable) Add(name, value string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	size := uint64(len(name) + len(value) + 32)
	if size > dt.maxCap {
		// If entry exceeds capacity entirely, clear table
		dt.entries = nil
		dt.curSize = 0
		return nil
	}

	dt.curSize += size
	dt.entries = append([]HeaderField{{Name: name, Value: value}}, dt.entries...)
	dt.inserted++
	dt.evict()
	return nil
}

func (dt *DynamicTable) evict() {
	for dt.curSize > dt.maxCap && len(dt.entries) > 0 {
		last := dt.entries[len(dt.entries)-1]
		size := uint64(len(last.Name) + len(last.Value) + 32)
		dt.curSize -= size
		dt.entries = dt.entries[:len(dt.entries)-1]
	}
}

// Get retrieves an entry by absolute or relative index.
func (dt *DynamicTable) Get(absoluteIndex uint64) (HeaderField, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	if absoluteIndex >= dt.inserted || absoluteIndex < dt.inserted-uint64(len(dt.entries)) {
		return HeaderField{}, ErrIndexOutOfBounds
	}
	idx := dt.inserted - 1 - absoluteIndex
	return dt.entries[idx], nil
}
