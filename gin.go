package httplog

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewGin wraps a new GinRouter for the gin router.
func NewGin(router *gin.Engine, store Store) *GinRouter {
	r := &GinRouter{
		Engine: router,
		mux:    NewMux(router, store),
	}
	fn := func(method string) RouterFn {
		return func(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouter {
			return r.Handle(method, relativePath, handler, options...)
		}
	}

	r.POST = fn(http.MethodPost)
	r.GET = fn(http.MethodGet)
	r.DELETE = fn(http.MethodDelete)
	r.PATCH = fn(http.MethodPatch)
	r.PUT = fn(http.MethodPut)
	r.OPTIONS = fn(http.MethodOptions)
	r.HEAD = fn(http.MethodHead)

	return r
}

// RouterFn defines the prototype for function gin Handle.
type RouterFn func(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouter

// GinRouterGroupFn defines the prototype for function gin Handle group.
type GinRouterGroupFn func(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouterGroup

// GinRouter defines adaptor routes implementation for IRoutes.
type GinRouter struct {
	*gin.Engine
	mux *Mux

	// XXX is a shortcut for router.Handle("XXX", path, handle).
	POST, GET, DELETE, PATCH, PUT, OPTIONS, HEAD RouterFn
}

// ServeHTTP calls f(w, r).
func (r *GinRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (r *GinRouter) Run(addr ...string) (err error) {
	address := resolveAddress(addr)
	err = http.ListenAndServe(address, r.mux)

	return
}

func resolveAddress(addr []string) string {
	switch len(addr) {
	case 0:
		if port := os.Getenv("PORT"); port != "" {
			return ":" + port
		}

		return ":8080"
	case 1:
		return addr[0]
	default:
		panic("too many parameters")
	}
}

// GinRouterGroup wraps the gin.RouterGroup.
type GinRouterGroup struct {
	*gin.RouterGroup
	GinRouter *GinRouter

	// XXX is a shortcut for router.Handle("XXX", path, handle).
	POST, GET, DELETE, PATCH, PUT, OPTIONS, HEAD GinRouterGroupFn
}

// Group creates a new router group. You should add all the routes that have common middlewares or the same path prefix.
// For example, all the routes that use a common middleware for authorization could be grouped.
func (r *GinRouter) Group(groupPath string, handlers ...gin.HandlerFunc) *GinRouterGroup {
	g := &GinRouterGroup{
		RouterGroup: r.Engine.Group(groupPath, handlers...),
		GinRouter:   r,
	}

	fn := func(method string) GinRouterGroupFn {
		return func(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouterGroup {
			g.Handle(method, relativePath, handler)
			r.mux.registerRouter(method, filepath.Join(groupPath, relativePath), options)

			return g
		}
	}

	g.POST = fn(http.MethodPost)
	g.GET = fn(http.MethodGet)
	g.DELETE = fn(http.MethodDelete)
	g.PATCH = fn(http.MethodPatch)
	g.PUT = fn(http.MethodPut)
	g.OPTIONS = fn(http.MethodOptions)
	g.HEAD = fn(http.MethodHead)

	return g
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware
// that can and should be shared among different routes.
// See the example code in GitHub.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *GinRouter) Handle(httpMethod, relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouter {
	r.Engine.Handle(httpMethod, relativePath, handler)
	r.mux.registerRouter(httpMethod, relativePath, options)

	return r
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (r *GinRouter) Any(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouter {
	r.Engine.Any(relativePath, handler)
	r.mux.registerRouter(anyMethod, relativePath, options)

	return r
}

// nolint:gochecknoglobals
var (
	ginHandlerFuncType = reflect.TypeOf((*gin.HandlerFunc)(nil)).Elem()
)

// RegisterCtler registers a controller object which declares the router in the structure fields' tag.
func (r *GinRouter) RegisterCtler(ctler interface{}) {
	v := reflect.ValueOf(ctler)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fi := t.Field(i)
		f := v.FieldByIndex(fi.Index)
		route := fi.Tag.Get("route")

		if f.IsNil() || route == "" || !fi.Type.AssignableTo(ginHandlerFuncType) {
			continue
		}

		method := "GET"

		if !strings.HasPrefix(route, "/") {
			pos := strings.Index(route, " ")
			method = route[:pos]
			route = strings.TrimSpace(route[pos+1:])
		}

		fn := f.Interface().(gin.HandlerFunc)
		options := []OptionFn{Biz(fi.Tag.Get("name")), Ignore(fi.Tag.Get("ignore") == "true")}

		if method == anyMethod {
			r.Any(route, fn, options...)
		} else {
			r.Handle(method, route, fn, options...)
		}
	}
}
