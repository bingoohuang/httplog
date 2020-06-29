package httplog

import (
	"net/http"
	"time"

	"github.com/bingoohuang/snow"

	"github.com/julienschmidt/httprouter"
)

// Mux defines the wrapper of http.ServeMux.
type Mux struct {
	handler   http.Handler
	router    *httprouter.Router
	store     Store
	muxOption *MuxOption
}

// ServeHTTP calls f(w, r).
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l := &Log{Created: time.Now()}
	holder := parseOption(r, mux)

	l.Option = holder.option
	l.PathParams = holder.params
	l.Biz = l.Option.GetBiz()

	l.Method = r.Method
	l.URL = r.URL.String()
	l.ReqHeader = r.Header
	l.Request = r

	if l.skipLoggingBefore(mux) {
		mux.handler.ServeHTTP(w, r)
		return
	}

	l.ID = snow.Next().String()
	l.IPAddr = GetRemoteAddress(r)
	l.ReqBody = string(PeekBody(r, maxSize))

	newCtx, ctxVar := createCtx(r, l)
	m := CaptureMetrics(mux.handler, w, r.WithContext(newCtx))

	l.RspStatus = m.Code
	l.RspBody = m.RespBody
	l.RespSize = m.Written
	l.Start = m.Start
	l.End = m.End
	l.Duration = m.Duration
	l.RspHeader = m.Header
	l.Attrs = ctxVar.Attrs

	if mux.store != nil {
		mux.store.Store(l)
	}
}

// HandlerFuncAware declares interface which holds  the HandleFunc function.
type HandlerFuncAware interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// MuxOption defines the option of mux.
type MuxOption struct {
	IgnoreBizNoname bool
}

// MuxOptionFn defines the function prototype to seting MuxOption.
type MuxOptionFn func(m *MuxOption)

// IgnoreBizNoname set the ignoreBizNoname option.
func IgnoreBizNoname(ignore bool) MuxOptionFn {
	return func(m *MuxOption) {
		m.IgnoreBizNoname = ignore
	}
}

// NewMux returns a new instance of Mux.
func NewMux(handler http.Handler, store Store, muxOptions ...MuxOptionFn) *Mux {
	muxOption := &MuxOption{}

	for _, fn := range muxOptions {
		fn(muxOption)
	}

	return &Mux{
		router:    httprouter.New(),
		handler:   handler,
		store:     store,
		muxOption: muxOption,
	}
}

// HandleFunc registers the handler function for the given pattern.
func (mux *Mux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), options ...OptionFn) {
	if v, ok := mux.handler.(HandlerFuncAware); ok {
		v.HandleFunc(pattern, handler)
	}

	mux.registerRouter(anyMethod, pattern, options)
}

// anyMethod means any HTTP method.
const anyMethod = "ANY"

// nolint:gochecknoglobals
var (
	allHTTPMethods = []string{
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
		if ww, ok := w.(*OptionHolder); ok {
			ww.option = option
			ww.params = p
		}
	}

	for _, m := range createMethods(method) {
		mux.router.Handle(m, pattern, f)
	}
}

func createMethods(method string) []string {
	if method == anyMethod {
		return allHTTPMethods
	}

	return []string{method}
}
