package httplog

import (
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

// HandleFunc registers the handler function for the given pattern.
func (mux *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), options ...OptionFn) {
	mux.ServeMux.HandleFunc(pattern, handler)
	mux.router.Handle("GET", pattern, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if ww, ok := w.(*optionsResponseWriter); ok {
			ww.options = options
		}
	})
}

// WrapHandler wraps a http.Handler for logging requests and responses.
func WrapHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		option := parseOption(r, h)

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
			IPAddr:      GetRemoteAddress(r),
			ReqHeader:   r.Header,
			ReqBody:     string(PeekBody(r, maxSize)),
		}

		newCtx, ctxVar := createCtx(r, ri)

		// this runs handler h and captures information about HTTP request
		m := CaptureMetrics(h, w, r.WithContext(newCtx))

		ri.RespCode = m.Code
		ri.RespBody = m.RespBody
		ri.RespSize = m.Written
		ri.Start = m.Start
		ri.End = m.End
		ri.Duration = m.Duration
		ri.RespHeader = m.Header
		ri.Attrs = ctxVar.Attrs

		logHTTPReq(ri)
	}

	return http.HandlerFunc(fn)
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
