package httplog

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/julienschmidt/httprouter"
)

// Mux defines the wrapper of http.ServeMux.
type Mux struct {
	handler http.Handler
	router  *httprouter.Router
}

// ServeHTTP calls f(w, r).
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	option := parseOption(r, mux)

	ri := &Req{
		HandlerName: option.GetName(),
		Method:      r.Method,
		URL:         r.URL.String(),

		ReqHeader: r.Header,
	}

	if skipLoggingBefore(ri, option) {
		mux.handler.ServeHTTP(w, r)
		return
	}

	ri.IPAddr = GetRemoteAddress(r)
	ri.ReqBody = string(PeekBody(r, maxSize))

	newCtx, ctxVar := createCtx(r, ri)
	m := CaptureMetrics(mux.handler, w, r.WithContext(newCtx))

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

// HandlerFuncAware declares interface which holds  the HandleFunc function.
type HandlerFuncAware interface {
	// HandleFunc registers the handler function for the given pattern.
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// NewMux returns a new instance of Mux.
func NewMux(handler http.Handler) *Mux {
	return &Mux{
		router:  httprouter.New(),
		handler: handler,
	}
}

// HandleFunc registers the handler function for the given pattern.
func (mux *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), options ...OptionFn) {
	if v, ok := mux.handler.(HandlerFuncAware); ok {
		v.HandleFunc(pattern, handler)
	}

	mux.registerRouter(AnyMethod, pattern, options)
}

// AnyMethod means any HTTP method.
const AnyMethod = "ANY"

// nolint:gochecknoglobals
var (
	AllHTTPMethods = []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
)

// registerRouter 记下路由，方便后面根据路由查找注册路由时的选项.
func (mux *Mux) registerRouter(method, pattern string, options []OptionFn) {
	option := (OptionFns(options)).CreateOption()
	f := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if ww, ok := w.(*optionsResponseWriter); ok {
			ww.option = option
		}
	}

	if method != AnyMethod {
		mux.router.Handle(method, pattern, f)

		return
	}

	for _, m := range AllHTTPMethods {
		mux.router.Handle(m, pattern, f)
	}
}
