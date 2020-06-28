package httplog_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/bingoohuang/httplog"
)

// from https://github.com/essentialbooks/books/blob/master/code/go/logging_http_requests/main.go
func ExampleNewServeMux() {
	mux := httplog.NewMux(http.NewServeMux(), &httplog.LogrusStore{})
	mux.HandleFunc("/echo", handleIndex, httplog.Name("回显处理"))
	mux.HandleFunc("/json", handleJSON, httplog.Name("JSON处理"))
	mux.HandleFunc("/ignored", handleIgnore, httplog.Ignore(true))
	mux.HandleFunc("/noname", handleNoname)

	r, _ := http.NewRequest("GET", "/json", nil)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body.String())

	r, _ = http.NewRequest("GET", "/echo", strings.NewReader(`{"name": "dingding"}`))
	r.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body.String())

	r, _ = http.NewRequest("GET", "/ignored", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body.String())

	r, _ = http.NewRequest("GET", "/noname", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	fmt.Println(w.Code, w.Body.String())

	// Output:
	// 202 {"name": "bingoohuang"}
	// 200 {"name": "dingding"}
	// 200 Ignored
	// 200 Noname
}

// simplest possible server that returns url as plain text.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	//msg := fmt.Sprintf("You've called url %s", r.URL.String())
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK) // 200

	attrs := httplog.ParseAttrs(r)
	attrs["bytes"] = "xxx"

	bytes, _ := ioutil.ReadAll(r.Body)
	_, _ = w.Write(bytes)
}

func handleIgnore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	_, _ = w.Write([]byte("Ignored"))
}

func handleNoname(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("Noname"))
}

func handleJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusAccepted) // 202
	_, _ = w.Write([]byte(`{"name": "bingoohuang"}`))
}
