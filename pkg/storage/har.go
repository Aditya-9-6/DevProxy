package storage

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// HAR represents the root HTTP Archive 1.2 container.
type HAR struct {
	Log HARLog `json:"log"`
}

type HARLog struct {
	Version string     `json:"version"`
	Creator HARCreator `json:"creator"`
	Entries []HAREntry `json:"entries"`
}

type HARCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HAREntry struct {
	StartedDateTime time.Time   `json:"startedDateTime"`
	Time            float64     `json:"time"` // duration in milliseconds
	Request         HARRequest  `json:"request"`
	Response        HARResponse `json:"response"`
	Cache           struct{}    `json:"cache"`
	Timings         HARTimings  `json:"timings"`
	ServerIPAddress string      `json:"serverIPAddress,omitempty"`
}

type HARRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []HARHeader  `json:"headers"`
	QueryString []HARQuery   `json:"queryString"`
	Cookies     []HARCookie  `json:"cookies"`
	HeaderSize  int          `json:"headersSize"`
	BodySize    int          `json:"bodySize"`
	PostData    *HARPostData `json:"postData,omitempty"`
}

type HARResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []HARHeader `json:"headers"`
	Cookies     []HARCookie `json:"cookies"`
	Content     HARContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeaderSize  int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

type HARHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HARQuery struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HARCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
}

type HARPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type HARContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

type HARTimings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// GenerateHAR converts stored RequestRecords into a standard HAR 1.2 JSON structure.
func GenerateHAR(records []*RequestRecord) ([]byte, error) {
	entries := make([]HAREntry, 0, len(records))

	for _, r := range records {
		reqHeaders := make([]HARHeader, 0)
		for k, vv := range r.ReqHeaders {
			for _, v := range vv {
				reqHeaders = append(reqHeaders, HARHeader{Name: k, Value: v})
			}
		}

		respHeaders := make([]HARHeader, 0)
		for k, vv := range r.RespHeaders {
			for _, v := range vv {
				respHeaders = append(respHeaders, HARHeader{Name: k, Value: v})
			}
		}

		reqMime := "application/octet-stream"
		if ct, ok := r.ReqHeaders["Content-Type"]; ok && len(ct) > 0 {
			reqMime = ct[0]
		}
		respMime := "text/plain"
		if ct, ok := r.RespHeaders["Content-Type"]; ok && len(ct) > 0 {
			respMime = ct[0]
		}

		var postData *HARPostData
		if len(r.ReqBody) > 0 {
			postData = &HARPostData{
				MimeType: reqMime,
				Text:     r.ReqBody,
			}
		}

		entry := HAREntry{
			StartedDateTime: r.Timestamp,
			Time:            r.DurationMs,
			ServerIPAddress: r.Host,
			Request: HARRequest{
				Method:      r.Method,
				URL:         r.URL,
				HTTPVersion: r.Proto,
				Headers:     reqHeaders,
				QueryString: extractQueryParams(r.URL),
				Cookies:     []HARCookie{},
				HeaderSize:  -1,
				BodySize:    len(r.ReqBody),
				PostData:    postData,
			},
			Response: HARResponse{
				Status:      r.StatusCode,
				StatusText:  "",
				HTTPVersion: r.Proto,
				Headers:     respHeaders,
				Cookies:     []HARCookie{},
				Content: HARContent{
					Size:     len(r.RespBody),
					MimeType: respMime,
					Text:     r.RespBody,
				},
				RedirectURL: "",
				HeaderSize:  -1,
				BodySize:    len(r.RespBody),
			},
			Timings: HARTimings{
				Send:    0,
				Wait:    r.DurationMs,
				Receive: 0,
			},
		}

		entries = append(entries, entry)
	}

	har := HAR{
		Log: HARLog{
			Version: "1.2",
			Creator: HARCreator{
				Name:    "DevProxy",
				Version: "1.0.0",
			},
			Entries: entries,
		},
	}

	return json.MarshalIndent(har, "", "  ")
}

func extractQueryParams(rawURL string) []HARQuery {
	queries := make([]HARQuery, 0)
	parts := strings.SplitN(rawURL, "?", 2)
	if len(parts) < 2 {
		return queries
	}
	pairs := strings.Split(parts[1], "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			queries = append(queries, HARQuery{Name: kv[0], Value: kv[1]})
		} else if len(kv) == 1 {
			queries = append(queries, HARQuery{Name: kv[0], Value: ""})
		}
	}
	return queries
}

// ParseHAR parses standard HAR 1.2 JSON structure into ringbuffer.TrafficEvent objects.
func ParseHAR(harBytes []byte) ([]*ringbuffer.TrafficEvent, error) {
	var har HAR
	if err := json.Unmarshal(harBytes, &har); err != nil {
		return nil, err
	}

	events := make([]*ringbuffer.TrafficEvent, 0, len(har.Log.Entries))
	for _, entry := range har.Log.Entries {
		reqHeaders := make(http.Header)
		for _, h := range entry.Request.Headers {
			reqHeaders.Add(h.Name, h.Value)
		}

		respHeaders := make(http.Header)
		for _, h := range entry.Response.Headers {
			respHeaders.Add(h.Name, h.Value)
		}

		var reqBody []byte
		if entry.Request.PostData != nil {
			reqBody = []byte(entry.Request.PostData.Text)
		}

		var respBody []byte
		if entry.Response.Content.Text != "" {
			respBody = []byte(entry.Response.Content.Text)
		}

		event := &ringbuffer.TrafficEvent{
			Timestamp:   entry.StartedDateTime,
			Duration:    time.Duration(entry.Time * float64(time.Millisecond)),
			ClientIP:    "127.0.0.1",
			Host:        entry.ServerIPAddress,
			Method:      entry.Request.Method,
			URL:         entry.Request.URL,
			Proto:       entry.Request.HTTPVersion,
			ReqHeaders:  reqHeaders,
			ReqBody:     reqBody,
			StatusCode:  entry.Response.Status,
			RespHeaders: respHeaders,
			RespBody:    respBody,
			TLS:         strings.HasPrefix(entry.Request.URL, "https"),
		}

		if event.Host == "" {
		    event.Host = reqHeaders.Get("Host")
		}

		events = append(events, event)
	}
	return events, nil
}
