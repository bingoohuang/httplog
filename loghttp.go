package httplog

import (
	"bufio"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// Mux defines the wrapper of http.ServeMux.
type Mux struct {
	*http.ServeMux
	router *httprouter.Router
}

// NewMux returns a new instance of Mux.
func NewMux() *Mux {
	return &Mux{ServeMux: &http.ServeMux{}, router: httprouter.New()}
}

// OptionFns defines the slice of OptionFns.
type OptionFns []OptionFn

type optionsResponseWriter struct{ options OptionFns }

func (ho optionsResponseWriter) Header() http.Header       { return http.Header{} }
func (ho optionsResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (ho optionsResponseWriter) WriteHeader(int)           {}

// Option defines the option for the handler in the httplog.
type Option struct {
	Name   string
	Ignore bool
}

// GetName returns the name from the option.
func (o Option) GetName() string {
	if o.Name != "" {
		return o.Name
	}

	return "Noname"
}

// CreateOption returns the option after functions call.
func (fns OptionFns) CreateOption() *Option {
	option := &Option{}

	for _, fn := range fns {
		fn(option)
	}

	return option
}

// OptionFn defines the option function prototype.
type OptionFn func(option *Option)

// Name defines the descriptive name of the handler.
func Name(name string) OptionFn { return func(option *Option) { option.Name = name } }

// Ignore tells the current handler should to be ignored for httplog.
func Ignore() OptionFn { return func(option *Option) { option.Ignore = true } }

// HandleFunc registers the handler function for the given pattern.
func (mux *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), options ...OptionFn) {
	mux.ServeMux.HandleFunc(pattern, handler)
	mux.router.Handle("GET", pattern, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if ww, ok := w.(*optionsResponseWriter); ok {
			ww.options = options
		}
	})
}

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

// ContextKey defines the context key type.
type ContextKey int

const (
	// CtxKey defines the context key for CtxVar.
	CtxKey ContextKey = iota
)

// ParseReq parses the Req from http.Request context.
func ParseReq(r *http.Request) *Req {
	if v, ok := r.Context().Value(CtxKey).(*CtxVar); ok {
		return v.Req
	}

	return &Req{}
}

// Attrs carries map. It implements Value for that key and
// delegates all other calls to the embedded Context.
type Attrs map[string]interface{}

// ParseAttrs returns the attributes map from the request context.
func ParseAttrs(r *http.Request) Attrs {
	if v, ok := r.Context().Value(CtxKey).(*CtxVar); ok {
		return v.Attrs
	}

	return Attrs{}
}

// CtxVar defines the context structure.
type CtxVar struct {
	Req   *Req
	Attrs Attrs
}

// WrapHandler wraps a http.Handler for logging requests and responses.
func WrapHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		option := parseHandlerOption(r, h)

		// websocket connections won't work when wrapped
		// in RecordingResponseWriter, so just pass those through
		if option.Ignore || IsWsRequest(r) {
			h.ServeHTTP(w, r)
			return
		}

		ri := &Req{
			HandlerName: option.GetName(),
			Method:      r.Method,
			URL:         r.URL.String(),
			Referer:     r.Header.Get("Referer"),
			UserAgent:   r.Header.Get("User-Agent"),
			ContentType: r.Header.Get("Content-Type"),
			IPAddr:      GetRemoteAddress(r),
			ReqBody:     string(peekBody(r)),
		}

		ctxVar := &CtxVar{Req: ri, Attrs: make(Attrs)}
		newCtx := context.WithValue(r.Context(), CtxKey, ctxVar)

		// this runs handler h and captures information about HTTP request
		m := CaptureMetrics(h, w, r.WithContext(newCtx))

		ri.RespCode = m.Code
		ri.RespBody = m.RespBody
		ri.RespSize = m.Written
		ri.Start = m.Start
		ri.End = m.End
		ri.Duration = m.Duration
		ri.Attrs = ctxVar.Attrs

		logHTTPReq(ri)
	}

	return http.HandlerFunc(fn)
}

func peekBody(r *http.Request) []byte {
	if r.Body == nil {
		return nil
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

	peek, _ := buf.Peek(maxSize)

	return peek
}

func parseHandlerOption(r *http.Request, h http.Handler) *Option {
	if mux, ok := h.(*Mux); ok {
		kw := &optionsResponseWriter{}
		mux.router.ServeHTTP(kw, r)

		return kw.options.CreateOption()
	}

	return &Option{Ignore: true}
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

	//srv.Addr = ":8100"
	//srv.ListenAndServe()

	return srv
}

// Req describes info about HTTP request.
type Req struct {
	HandlerName string

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
	Attrs    Attrs
}

// we mostly care page views. to log less we skip logging
// of urls that don't provide useful information. hopefully we won't regret it.
func skipLogging(ri *Req) bool {
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

func logHTTPReq(ri *Req) {
	if skipLogging(ri) {
		return
	}

	logrus.Infof("http:%+v\n", ri)
}
