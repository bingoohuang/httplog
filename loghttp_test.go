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
func ExampleWrapHandler() {
	mux := &http.ServeMux{}
	mux.HandleFunc("/echo", handleIndex)
	mux.HandleFunc("/json", handleJSON)

	//httpSrv := &http.Server{
	//	ReadTimeout:  120 * time.Second,
	//	WriteTimeout: 120 * time.Second,
	//	IdleTimeout:  120 * time.Second, // introduced in Go 1.8
	//	Handler:     httplog.WrapHandler(mux),
	//}
	//
	//httpSrv.Addr = ":8100"
	//
	//fmt.Println(httpSrv.Addr)
	//httpSrv.ListenAndServe()

	r, _ := http.NewRequest("GET", "/json", nil)
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	httplog.WrapHandler(mux).ServeHTTP(w, r)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())

	r, _ = http.NewRequest("GET", "/echo", strings.NewReader(`{"name": "dingding"}`))
	r.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	httplog.WrapHandler(mux).ServeHTTP(w, r)

	fmt.Println(w.Code)
	fmt.Println(w.Body.String())

	// Output:
	// 202
	// {"name": "bingoohuang"}
	// 200
	// {"name": "dingding"}
}

// simplest possible server that returns url as plain text.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	//msg := fmt.Sprintf("You've called url %s", r.URL.String())
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK) // 200

	bytes, _ := ioutil.ReadAll(r.Body)
	_, _ = w.Write(bytes)
}

func handleJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusAccepted) // 202
	_, _ = w.Write([]byte(`{"name": "bingoohuang"}`))
}
