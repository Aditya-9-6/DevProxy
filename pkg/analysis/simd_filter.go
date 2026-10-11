package analysis

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

var (
	sqliMarkers = [][]byte{
		[]byte("SELECT "), []byte("UNION "), []byte("INSERT "), []byte("DELETE "),
		[]byte("DROP "), []byte("UPDATE "), []byte("OR 1=1"), []byte("--"),
	}
	xssMarkers = [][]byte{
		[]byte("<script"), []byte("javascript:"), []byte("onerror="), []byte("onload="),
		[]byte("alert("), []byte("eval("),
	}
)

func ContainsSecurityMarker(data []byte) bool {
	for _, m := range sqliMarkers {
		if bytes.Contains(data, m) {
			return true
		}
	}
	for _, m := range xssMarkers {
		if bytes.Contains(data, m) {
			return true
		}
	}
	return false
}
