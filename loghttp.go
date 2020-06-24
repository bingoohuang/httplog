package httplog

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Request.RemoteAddress contains port, which we want to remove i.e.:
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// requestGetRemoteAddress returns ip address of the client making the request,
// taking into account http proxies
func requestGetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIP == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}

// return true if this request is a websocket request
func isWsRequest(r *http.Request) bool {
	uri := r.URL.Path
	return strings.HasPrefix(uri, "/ws/")
}

func logRequestHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// websocket connections won't work when wrapped
		// in RecordingResponseWriter, so just pass those through
		if isWsRequest(r) {
			h.ServeHTTP(w, r)
			return
		}

		buf := bufio.NewReader(r.Body)
		// And now set a new body, which will simulate the same data we read:
		r.Body = ioutil.NopCloser(buf)

		// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
		// Use http.MaxBytesReader to enforce a maximum read of 1MB from the
		// response body. A request body larger than that will now result in
		// Decode() returning a "http: request body too large" error.
		// r.Body = http.MaxBytesReader(w, r.Body, 1048576)

		// Work / inspect body. You may even modify it!

		peek, _ := buf.Peek(10)
		ri := &HTTPReq{
			method:      r.Method,
			url:         r.URL.String(),
			referer:     r.Header.Get("Referer"),
			userAgent:   r.Header.Get("User-Agent"),
			contentType: r.Header.Get("Content-Type"),
			reqBody:     string(peek),
		}

		ri.ipaddr = requestGetRemoteAddress(r)

		// this runs handler h and captures information about
		// HTTP request
		m := CaptureMetrics(h, w, r)

		ri.code = m.Code
		ri.respBody = m.RespBody
		ri.size = m.Written
		ri.duration = m.Duration
		logHTTPReq(ri)
	}
	return http.HandlerFunc(fn)
}

func MakeHTTPServer(h http.Handler) *http.Server {
	srv := &http.Server{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      logRequestHandler(h),
	}
	return srv
}

// HTTPReq describes info about HTTP request
type HTTPReq struct {
	// GET etc.
	method      string
	url         string
	reqBody     string
	referer     string
	contentType string
	ipaddr      string
	// response code, like 200, 404
	code     int
	respBody string
	// number of bytes of the response sent
	size int64
	// how long did it take to
	duration  time.Duration
	userAgent string
}

// we mostly care page views. to log less we skip logging
// of urls that don't provide useful information.
// hopefully we won't regret it
func skipHTTPRequestLogging(ri *HTTPReq) bool {
	// we always want to know about failures and other
	// non-200 responses
	if ri.code != 200 {
		return false
	}

	// we want to know about slow requests.
	// 100 ms threshold is somewhat arbitrary
	if ri.duration > 100*time.Millisecond {
		return false
	}

	// this is linked from every page
	if ri.url == "/favicon.png" {
		return true
	}

	if ri.url == "/favicon.ico" {
		return true
	}

	if strings.HasSuffix(ri.url, ".css") {
		return true
	}

	return false
}

func logHTTPReq(ri *HTTPReq) {
	if skipHTTPRequestLogging(ri) {
		return
	}

	fmt.Printf("loghttp:%+v\n", ri)
}
