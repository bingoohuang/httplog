package httplog

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/bingoohuang/snow"
	"github.com/sirupsen/logrus"
	"github.com/spyzhov/ajson"
)

type colFn func(log *Log) interface{}

type col interface {
	get(log *Log) interface{}
}

func (f colFn) get(log *Log) interface{} { return f(log) }

type colVFn func(log *Log, v string) interface{}

type colV interface {
	get(log *Log, v string) interface{}
}

func (f colVFn) get(log *Log, v string) interface{} { return f(log, v) }

// nolint:gochecknoglobals
var (
	tagPattern = regexp.MustCompile(`httplog:"(.*?)"`)

	blts = make(map[matcher]col)
	rsps = make(map[matcher]colV)
	reqs = make(map[matcher]colV)
)

func getJSONBody(contentType, body string) string {
	if !strings.Contains(contentType, "json") {
		return ""
	}

	if strings.HasPrefix(body, "{") || strings.HasPrefix(body, "[") {
		return body
	}

	return ""
}

func jsonpath(expr, body string) string {
	path := expr
	if !strings.HasPrefix(expr, "$.") {
		path = "$." + expr
	}

	nodes, err := ajson.JSONPath([]byte(body), path)
	if err != nil {
		logrus.Warnf("failed to eval JSONPath %s for body %s error %+v", path, body, err)
		return ""
	}

	if len(nodes) == 1 {
		return fmt.Sprintf("%+v", nodes[0])
	}

	return fmt.Sprintf("%+v", nodes)
}

// nolint:lll,gochecknoinits
func init() {
	blts[eq("id")] = colFn(func(l *Log) interface{} { return l.ID })
	blts[eq("created")] = colFn(func(l *Log) interface{} { return l.Created })
	blts[eq("ip")] = colFn(func(l *Log) interface{} { return snow.InferHostIPv4("") })
	blts[eq("hostname")] = colFn(func(l *Log) interface{} { v, _ := os.Hostname(); return v })
	blts[eq("pid")] = colFn(func(l *Log) interface{} { return os.Getpid() })
	blts[eq("started")] = colFn(func(l *Log) interface{} { return l.Start })
	blts[eq("end")] = colFn(func(l *Log) interface{} { return l.End })
	blts[eq("cost")] = colFn(func(l *Log) interface{} { return l.Duration.Milliseconds() })
	blts[eq("biz")] = colFn(func(l *Log) interface{} { return l.Biz })

	rsps[starts("head_")] = colVFn(func(l *Log, v string) interface{} { return At(l.RspHeader[v[5:]], 0) })
	rsps[eq("heads")] = colVFn(func(l *Log, v string) interface{} { return fmt.Sprintf("%+v", l.RspHeader) })
	rsps[eq("body")] = colVFn(func(l *Log, v string) interface{} { return l.RspBody })
	rsps[eq("json")] = colVFn(func(l *Log, v string) interface{} { return getJSONBody(At(l.RspHeader["Content-Type"], 0), l.RspBody) })
	rsps[starts("json_")] = colVFn(func(l *Log, v string) interface{} { return jsonpath(v[5:], l.RspBody) })
	rsps[eq("status")] = colVFn(func(l *Log, v string) interface{} { return l.RspStatus })

	reqs[starts("head_")] = colVFn(func(l *Log, v string) interface{} { return At(l.ReqHeader[v[5:]], 0) })
	reqs[eq("heads")] = colVFn(func(l *Log, v string) interface{} { return fmt.Sprintf("%+v", l.ReqHeader) })
	reqs[eq("body")] = colVFn(func(l *Log, v string) interface{} { return l.ReqBody })
	reqs[eq("json")] = colVFn(func(l *Log, v string) interface{} { return getJSONBody(At(l.ReqHeader["Content-Type"], 0), l.ReqBody) })
	reqs[starts("json_")] = colVFn(func(l *Log, v string) interface{} { return jsonpath(v[5:], l.ReqBody) })

	reqs[eq("method")] = colVFn(func(l *Log, v string) interface{} { return l.Method })
	reqs[eq("url")] = colVFn(func(l *Log, v string) interface{} { return l.URL })
	reqs[starts("path_")] = colVFn(func(l *Log, v string) interface{} { return l.pathVar(v[5:]) })
	reqs[eq("paths")] = colVFn(func(l *Log, v string) interface{} { return l.pathVars() })
	reqs[starts("query_")] = colVFn(func(l *Log, v string) interface{} { return l.queryVar(v[6:]) })
	reqs[eq("queries")] = colVFn(func(l *Log, v string) interface{} { return l.queryVars() })
	reqs[starts("param_")] = colVFn(func(l *Log, v string) interface{} { return l.paramVar(v[6:]) })
	reqs[eq("params")] = colVFn(func(l *Log, v string) interface{} { return l.paramVars() })
}

func (s *TableCol) parseComment() {
	tag := strings.ToLower(s.Name)
	if tag != "" {
		sub := tagPattern.FindAllStringSubmatch(s.Comment, 1)
		if len(sub) > 0 {
			tag = sub[0][1]
		}
	}

	switch {
	case strings.HasPrefix(tag, "req_"):
		s.ValueGetter = createValueGetter(tag[4:], reqs)
	case strings.HasPrefix(tag, "rsp_"):
		s.ValueGetter = createValueGetter(tag[4:], rsps)
	case strings.HasPrefix(tag, "ctx_"):
		s.ValueGetter = createCtxValueGetter(tag[4:])
	case tag == "-":
		s.ValueGetter = nil
	default:
		s.ValueGetter = createBuiltinValueGetter(tag)
	}

	if s.ValueGetter != nil {
		s.ValueGetter = s.wrapMaxLength(s.ValueGetter)
	}
}

func (s *TableCol) wrapMaxLength(col col) col {
	return colFn(func(l *Log) interface{} {
		v := col.get(l)

		if v == nil || s.MaxLength <= 0 {
			return v
		}

		switch reflect.TypeOf(v).String() {
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"bool",
			"float32", "float64",
			"time.Time":
			return v
		}

		return Abbreviate(fmt.Sprintf("%v", v), s.MaxLength)
	})
}

func findGetterV(tag string, m map[matcher]colV) colV {
	for k, v := range m {
		if k.matches(tag) {
			return v
		}
	}

	return nil
}
func findGetter(tag string, m map[matcher]col) col {
	for k, v := range m {
		if k.matches(tag) {
			return v
		}
	}

	return nil
}

type v struct {
	colV
	v string
}

func (v v) get(log *Log) interface{} {
	return v.colV.get(log, v.v)
}

func createBuiltinValueGetter(tag string) col {
	return findGetter(tag, blts)
}

func createCtxValueGetter(tag string) col {
	return colFn(func(l *Log) interface{} {
		return l.Attrs[tag]
	})
}

func createValueGetter(tag string, m map[matcher]colV) col {
	getterV := findGetterV(tag, m)
	if getterV == nil {
		return nil
	}

	return &v{colV: getterV, v: tag}
}

type matcher interface {
	matches(tag string) bool
}

type equalMatcher struct {
	value string
}

func eq(v string) matcher {
	return equalMatcher{value: v}
}

func (r equalMatcher) matches(tag string) bool {
	return r.value == tag
}

type startsMatcher struct {
	Value string
}

func (r startsMatcher) matches(tag string) bool {
	return strings.HasPrefix(tag, r.Value)
}

func starts(v string) matcher {
	return startsMatcher{Value: v}
}
