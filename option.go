package httplog

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// OptionFns defines the slice of OptionFns.
type OptionFns []OptionFn

// OptionHolder defines the holder for option and params.
type OptionHolder struct {
	option *Option
	params httprouter.Params
}

// Header returns the header map that will be sent by WriteHeader.
func (ho OptionHolder) Header() http.Header { return http.Header{} }

// Write writes the data to the connection as part of an HTTP reply.
func (ho OptionHolder) Write([]byte) (int, error) { return 0, nil }

// WriteHeader sends an HTTP response header with the provided status code.
func (ho OptionHolder) WriteHeader(int) {}

// Option defines the option for the handler in the httplog.
type Option struct {
	Biz    string
	Tables []string
	Ignore bool
}

func parseOption(r *http.Request, h http.Handler) *OptionHolder {
	mux, ok := h.(*Mux)
	kw := &OptionHolder{option: &Option{Ignore: true}}

	if !ok {
		return kw
	}

	mux.router.ServeHTTP(kw, r)

	return kw
}

// GetBiz returns the name from the option.
func (o Option) GetBiz() string {
	if o.Biz != "" {
		return o.Biz
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

// Biz defines the descriptive name of the handler.
func Biz(name string) OptionFn { return func(option *Option) { option.Biz = name } }

// Tables defines the tables to saving log.
func Tables(names ...string) OptionFn { return func(option *Option) { option.Tables = names } }

// Ignore tells the current handler should to be ignored for httplog.
func Ignore(ignore bool) OptionFn {
	return func(option *Option) {
		option.Ignore = ignore
	}
}
