package httplog

import (
	"context"
	"net/http"
)

// ContextKey defines the context key type.
type ContextKey int

const (
	// CtxKey defines the context key for CtxVar.
	CtxKey ContextKey = iota
)

// CtxVar defines the context structure.
type CtxVar struct {
	Req   *Log
	Attrs Attrs
}

// ParseReq parses the Log from http.Request context.
func ParseReq(r *http.Request) *Log {
	if v, ok := r.Context().Value(CtxKey).(*CtxVar); ok {
		return v.Req
	}

	return &Log{}
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

func createCtx(r *http.Request, ri *Log) (context.Context, *CtxVar) {
	ctxVar := &CtxVar{Req: ri, Attrs: make(Attrs)}
	newCtx := context.WithValue(r.Context(), CtxKey, ctxVar)

	return newCtx, ctxVar
}
