package httplog

import (
	"net/http"

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

		ri := &Req{
			HandlerName: option.GetName(),
			Method:      r.Method,
			URL:         r.URL.String(),

			ReqHeader: r.Header,
		}

		if skipLoggingBefore(ri, option) {
			h.ServeHTTP(w, r)
			return
		}

		ri.IPAddr = GetRemoteAddress(r)
		ri.ReqBody = string(PeekBody(r, maxSize))

		newCtx, ctxVar := createCtx(r, ri)
		m := CaptureMetrics(h, w, r.WithContext(newCtx))

		ri.RespCode = m.Code
		ri.RespBody = m.RespBody
		ri.RespSize = m.Written
		ri.Start = m.Start
		ri.End = m.End
		ri.Duration = m.Duration
		ri.RespHeader = m.Header
		ri.Attrs = ctxVar.Attrs

		logrus.Infof("http:%+v\n", ri)
	}

	return http.HandlerFunc(fn)
}
