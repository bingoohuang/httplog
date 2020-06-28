package httplog

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// RequestVars defines the structure of request vars tha can be set.
type RequestVars struct {
	Body        io.Reader
	ContentType string
}

// RequestVarsFn defines the prototype of RequestVars option setting function.
type RequestVarsFn func(r *RequestVars)

// RequestVarsFns is the slice of RequestVarsFn.
type RequestVarsFns []RequestVarsFn

// Create creates new RequestVars.
func (fns RequestVarsFns) Create() *RequestVars {
	vars := &RequestVars{}

	for _, fn := range fns {
		fn(vars)
	}

	return vars
}

// JSONVar creates a new JSON RequestVarsFn.
func JSONVar(obj interface{}) RequestVarsFn {
	return func(r *RequestVars) {
		if s, ok := obj.(string); ok {
			r.Body = strings.NewReader(s)
		} else {
			b, _ := JSONMarshal(obj)
			r.Body = bytes.NewReader(b)
		}

		r.ContentType = "application/json; charset=utf-8"
	}
}

// PerformRequest performs a test request.
// from https://github.com/gin-gonic/gin/issues/1120.
func PerformRequest(method, target string, fn http.Handler, fns ...RequestVarsFn) *httptest.ResponseRecorder {
	vars := (RequestVarsFns(fns)).Create()

	r := httptest.NewRequest(method, target, vars.Body)

	if vars.ContentType != "" {
		r.Header.Set("Content-Type", vars.ContentType)
	}

	w := httptest.NewRecorder()
	fn.ServeHTTP(w, r)

	return w
}
