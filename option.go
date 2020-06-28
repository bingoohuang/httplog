package httplog

import "net/http"

// OptionFns defines the slice of OptionFns.
type OptionFns []OptionFn

type optionsResponseWriter struct{ option *Option }

func (ho optionsResponseWriter) Header() http.Header       { return http.Header{} }
func (ho optionsResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (ho optionsResponseWriter) WriteHeader(int)           {}

// Option defines the option for the handler in the httplog.
type Option struct {
	Name   string
	Ignore bool
}

func parseOption(r *http.Request, h http.Handler) *Option {
	mux, ok := h.(*Mux)

	if !ok {
		return &Option{Ignore: true}
	}

	kw := &optionsResponseWriter{}

	mux.router.ServeHTTP(kw, r)

	return kw.option
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
func Ignore(ignore bool) OptionFn {
	return func(option *Option) {
		option.Ignore = ignore
	}
}
