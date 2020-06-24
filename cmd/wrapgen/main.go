package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"strings"
)

type Build struct {
	Suffix     string
	Tags       string
	Interfaces Interfaces
}

func (b *Build) MustBuild() {
	prefix := "wrap_generated_"
	b.Implementation().MustWriteFile(prefix + b.Suffix + ".go")
	b.Tests().MustWriteFile(prefix + b.Suffix + "_test.go")
}

func (b *Build) writeHeader(g *Generator) {
	g.Printf(`
// +build %s
// Code generated by "httpsnoop/codegen"; DO NOT EDIT
package httpsnoop
`, b.Tags)
}

func (b *Build) Implementation() *Generator {
	ifaces := b.Interfaces
	// subIfaces has all interfaces except http.ResponseWriter
	subIfaces := ifaces[1:]

	var g Generator
	// Package header
	b.writeHeader(&g)
	g.Printf("import (\n")
	g.Printf(`"net/http"` + "\n")
	g.Printf(`"io"` + "\n")
	g.Printf(`"net"` + "\n")
	g.Printf(`"bufio"` + "\n")
	g.Printf(")\n")
	g.Printf("\n")

	// Hook funcs
	for _, iface := range ifaces {
		for _, fn := range iface.Funcs {
			g.Printf("// %s is part of the %s interface.\n", fn.Type(), iface.Name)
			g.Printf("type %s func(%s) (%s)\n", fn.Type(), fn.Args, fn.Returns)
			g.Printf("\n")
		}
	}

	// Hooks struct
	g.Printf(`
// Hooks defines a set of method interceptors for methods included in
// http.ResponseWriter as well as some others. You can think of them as
// middleware for the function calls they target. See Wrap for more details.
type Hooks struct {
`)
	for _, iface := range ifaces {
		for _, fn := range iface.Funcs {
			g.Printf("%s func(%s) %s\n", fn.Name, fn.Type(), fn.Type())
		}
	}
	g.Printf("}\n")

	// Wrap func
	docList := make([]string, len(subIfaces))
	for i, iface := range subIfaces {
		docList[i] = "// - " + iface.Name
	}
	g.Printf(`
// Wrap returns a wrapped version of w that provides the exact same interface
// as w. Specifically if w implements any combination of:
// 
%s
//
// The wrapped version will implement the exact same combination. If no hooks
// are set, the wrapped version also behaves exactly as w. Hooks targeting
// methods not supported by w are ignored. Any other hooks will intercept the
// method they target and may modify the call's arguments and/or return values.
// The CaptureMetrics implementation serves as a working example for how the
// hooks can be used.
`, strings.Join(docList, "\n"))
	g.Printf("func Wrap(w http.ResponseWriter, hooks Hooks) http.ResponseWriter {\n")
	g.Printf("rw := &rw{w: w, h: hooks}\n")
	for i, iface := range subIfaces {
		g.Printf("_, i%d := w.(%s)\n", i, iface.Name)
	}
	g.Printf("switch {\n")
	combinations := 1 << uint(len(subIfaces))
	for i := 0; i < combinations; i++ {
		conditions := make([]string, len(subIfaces))
		fields := make([]string, 0, len(subIfaces))
		fields = append(fields, "http.ResponseWriter")
		for j, iface := range subIfaces {
			ok := i&(1<<uint(len(subIfaces)-j-1)) > 0
			if !ok {
				conditions[j] = "!"
			} else {
				fields = append(fields, iface.Name)
			}
			conditions[j] += fmt.Sprintf("i%d", j)
		}
		values := make([]string, len(fields))
		for i := range fields {
			values[i] = "rw"
		}
		g.Printf("// combination %d/%d\n", i+1, combinations)
		g.Printf("case %s:\n", strings.Join(conditions, "&&"))
		fieldsS, valuesS := strings.Join(fields, "\n"), strings.Join(values, ",")
		g.Printf("return struct{\n%s\n}{%s}\n", fieldsS, valuesS)
	}
	g.Printf("}\n")
	g.Printf("panic(\"unreachable\")")
	g.Printf("}\n")

	// rw struct
	g.Printf(`
type rw struct {
	w http.ResponseWriter
	h Hooks
}
`)
	for _, iface := range ifaces {
		for _, fn := range iface.Funcs {
			g.Printf("func (w *rw) %s(%s) (%s) {\n", fn.Name, fn.Args, fn.Returns)
			g.Printf("f := w.w.(%s).%s\n", iface.Name, fn.Name)
			g.Printf("if w.h.%s != nil {\n", fn.Name)
			g.Printf("f = w.h.%s(f)\n", fn.Name)
			g.Printf("}\n")
			if fn.Returns != "" {
				g.Printf("return ")
			}
			g.Printf("f(%s)\n", fn.Args.Names())
			g.Printf("}\n")
			g.Printf("\n")
		}
	}
	return &g
}

func (b *Build) Tests() *Generator {
	ifaces := b.Interfaces
	// @TODO dedupe
	// subIfaces has all interfaces except http.ResponseWriter
	subIfaces := ifaces[1:]

	var g Generator
	// Package header
	b.writeHeader(&g)
	g.Printf("import (\n")
	g.Printf(`"net/http"` + "\n")
	g.Printf(`"io"` + "\n")
	g.Printf(`"testing"` + "\n")
	g.Printf(")\n")
	g.Printf("\n")

	// TestWrap func
	g.Printf("func TestWrap(t *testing.T) {\n")
	combinations := 1 << uint(len(subIfaces))
	for i := 0; i < combinations; i++ {
		fields := make([]string, 0, len(subIfaces))
		fields = append(fields, "http.ResponseWriter")
		expected := make([]bool, len(ifaces))
		expected[0] = true
		for j, iface := range subIfaces {
			ok := i&(1<<uint(len(subIfaces)-j-1)) > 0
			expected[j+1] = ok
			if ok {
				fields = append(fields, iface.Name)
			}
		}
		g.Printf("// combination %d/%d\n", i+1, combinations)
		g.Printf("{\n")
		g.Printf(`t.Log("%s")`+"\n", strings.Join(fields, ", "))
		g.Printf("w := Wrap(struct{\n%s\n}{}, Hooks{})\n", strings.Join(fields, "\n"))
		for i, iface := range ifaces {
			g.Printf("if _, ok := w.(%s); ok != %t {\n", iface.Name, expected[i])
			g.Printf("t.Error(\"unexpected interface\");\n")
			g.Printf("}\n")
		}
		g.Printf("}\n")
		g.Printf("\n")
	}
	g.Printf("}\n")
	return &g
}

type Interfaces []*Interface

type Interface struct {
	Name  string
	Funcs []*InterfaceFunc
}

type InterfaceFunc struct {
	Name    string
	Args    FuncArgs
	Returns string
}

type FuncArgs []*FuncArg

func (fa FuncArgs) String() string {
	args := make([]string, len(fa))
	for i, a := range fa {
		args[i] = a.Name + " " + a.Type
	}
	return strings.Join(args, ", ")
}

func (fa FuncArgs) Names() string {
	args := make([]string, len(fa))
	for i, a := range fa {
		args[i] = a.Name
	}
	return strings.Join(args, ", ")
}

type FuncArg struct {
	Name string
	Type string
}

func (fn *InterfaceFunc) Type() string {
	return fn.Name + "Func"
}

type Generator struct {
	buf bytes.Buffer
}

func (g *Generator) Printf(s string, args ...interface{}) {
	fmt.Fprintf(&g.buf, s, args...)
}

func (g *Generator) WriteFile(name string) error {
	src, err := g.Format()
	if err != nil {
		return fmt.Errorf("format: %s: %s:\n\n%s\n", name, err, g.Bytes())
	} else if err := ioutil.WriteFile(name, src, 0644); err != nil {
		return err
	}
	return nil

}

func (g *Generator) MustWriteFile(name string) {
	if err := g.WriteFile(name); err != nil {
		fatalf("%s", err)
	}
}

func (g *Generator) Bytes() []byte {
	return g.buf.Bytes()
}

func (g *Generator) Format() ([]byte, error) {
	return format.Source(g.Bytes())
}

func main() {
	ifaces := Interfaces{
		{
			Name: "http.ResponseWriter",
			Funcs: []*InterfaceFunc{
				{"Header", nil, "http.Header"},
				{"WriteHeader", FuncArgs{{"code", "int"}}, ""},
				{"Write", FuncArgs{{"b", "[]byte"}}, "int, error"},
			},
		},
		{
			Name: "http.Flusher",
			Funcs: []*InterfaceFunc{
				{"Flush", nil, ""},
			},
		},
		{
			Name: "http.CloseNotifier",
			Funcs: []*InterfaceFunc{
				{"CloseNotify", nil, "<-chan bool"},
			},
		},
		{
			Name: "http.Hijacker",
			Funcs: []*InterfaceFunc{
				{"Hijack", nil, "net.Conn, *bufio.ReadWriter, error"},
			},
		},
		{
			Name: "io.ReaderFrom",
			Funcs: []*InterfaceFunc{
				{"ReadFrom", FuncArgs{{"src", "io.Reader"}}, "int64, error"},
			},
		},
	}
	builds := []Build{
		{
			Suffix:     "lt_1.8",
			Tags:       "!go1.8",
			Interfaces: ifaces,
		},
		{
			Suffix: "gteq_1.8",
			Tags:   "go1.8",
			Interfaces: append(ifaces, &Interface{
				Name: "http.Pusher",
				Funcs: []*InterfaceFunc{
					{"Push", FuncArgs{
						{"target", "string"},
						{"opts", "*http.PushOptions"},
					}, "error"},
				},
			}),
		},
	}
	for _, build := range builds {
		build.MustBuild()
	}
}

func fatalf(s string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, s+"\n", args...)
	os.Exit(1)
}
