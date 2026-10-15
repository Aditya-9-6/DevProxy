package waf

import (
	"github.com/Aditya-9-6/DevProxy/pkg/internal/align"
)

// CacheLinePad prevents false sharing by using the shared alignment type.
type CacheLinePad = align.CacheLinePad
