package httplog

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// IPAddrFromRemoteAddr parses the IP Address.
// Request.RemoteAddress contains port, which we want to remove i.e.: "[::1]:58292" => "[::1]".
func IPAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}

	return s[:idx]
}

// GetRemoteAddress returns ip address of the client making the request, taking into account http proxies.
func GetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")

	if hdrRealIP == "" && hdrForwardedFor == "" {
		return IPAddrFromRemoteAddr(r.RemoteAddr)
	}

	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}

		return parts[0]
	}

	return hdrRealIP
}

// IsWsRequest return true if this request is a websocket request.
func IsWsRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/ws/")
}

// WrapHandler wraps a http.Handler for logging requests and responses.
func WrapHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// websocket connections won't work when wrapped
		// in RecordingResponseWriter, so just pass those through
		if IsWsRequest(r) {
			h.ServeHTTP(w, r)
			return
		}

		var peek []byte

		if r.Body != nil {
			buf := bufio.NewReader(r.Body)
			// And now set a new body, which will simulate the same data we read:
			r.Body = ioutil.NopCloser(buf)

			// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
			// Use http.MaxBytesReader to enforce a maximum read of 1MB from the
			// response body. A request body larger than that will now result in
			// Decode() returning a "http: request body too large" error.
			// r.Body = http.MaxBytesReader(w, r.Body, 1048576)

			// Work / inspect body. You may even modify it!

			peek, _ = buf.Peek(maxSize)
		}

		ri := &HTTPReq{
			Method:      r.Method,
			URL:         r.URL.String(),
			Referer:     r.Header.Get("Referer"),
			UserAgent:   r.Header.Get("User-Agent"),
			ContentType: r.Header.Get("Content-Type"),
			ReqBody:     string(peek),
		}

		ri.IPAddr = GetRemoteAddress(r)

		// this runs handler h and captures information about HTTP request
		m := CaptureMetrics(h, w, r)

		ri.RespCode = m.Code
		ri.RespBody = m.RespBody
		ri.RespSize = m.Written
		ri.Start = m.Start
		ri.End = m.End
		ri.Duration = m.Duration

		logHTTPReq(ri)
	}

	return http.HandlerFunc(fn)
}

// WrapServer wraps a http server with log http wrapped handler.
// nolint gomnd
func WrapServer(h http.Handler) *http.Server {
	srv := &http.Server{
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second, // introduced in Go 1.8
		Handler:      WrapHandler(h),
	}
	return srv
}

// HTTPReq describes info about HTTP request.
type HTTPReq struct {
	// Method is GET etc.
	Method      string
	URL         string
	Referer     string
	UserAgent   string
	ContentType string
	IPAddr      string
	ReqBody     string

	// RespCode, like 200, 404
	RespCode int
	// RespSize is number of bytes of the response sent
	RespSize int64
	// RespBody is the response body(limit to 1000)
	RespBody string

	// Start records the start time of the request
	Start time.Time
	// End records the end time of the request
	End time.Time
	// Duration means how long did it take to
	Duration time.Duration
}

// we mostly care page views. to log less we skip logging
// of urls that don't provide useful information. hopefully we won't regret it.
func skipLogging(ri *HTTPReq) bool {
	// we always want to know about failures and other non-2xx responses
	if !(ri.RespCode >= 200 && ri.RespCode < 300) {
		return false
	}

	// we want to know about slow requests. 100 ms threshold is somewhat arbitrary
	if ri.Duration > 100*time.Millisecond {
		return false
	}

	// this is linked from every page
	if ri.URL == "/favicon.png" || ri.URL == "/favicon.ico" {
		return true
	}

	if strings.HasSuffix(ri.URL, ".css") {
		return true
	}

	return false
}

func logHTTPReq(ri *HTTPReq) {
	if skipLogging(ri) {
		return
	}

	logrus.Infof("http:%+v\n", ri)
}
