package httplog

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// NewGin wraps a new GinRouter for the gin router.
func NewGin(router gin.IRouter, store Store) *GinRouter {
	r := &GinRouter{
		IRouter: router,
		Mux:     NewMux(router.(http.Handler), store),
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

// GinRouter defines adaptor routes implementation for IRoutes.
type GinRouter struct {
	gin.IRouter
	*Mux

	// XXX is a shortcut for router.Handle("XXX", path, handle).
	POST, GET, DELETE, PATCH, PUT, OPTIONS, HEAD RouterFn
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
	r.IRouter.Handle(httpMethod, relativePath, handler)
	r.registerRouter(httpMethod, relativePath, options)

	return r
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (r *GinRouter) Any(relativePath string, handler gin.HandlerFunc, options ...OptionFn) *GinRouter {
	r.IRouter.Any(relativePath, handler)
	r.registerRouter(AnyMethod, relativePath, options)

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
		options := []OptionFn{Name(fi.Tag.Get("name")), Ignore(fi.Tag.Get("ignore") == "true")}

		if method == AnyMethod {
			r.Any(route, fn, options...)
		} else {
			r.Handle(method, route, fn, options...)
		}
	}
}
