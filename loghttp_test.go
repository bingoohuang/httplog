package httplog_test

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bingoohuang/httplog"
)

// from https://github.com/essentialbooks/books/blob/master/code/go/logging_http_requests/main.go
func ExampleMakeHTTPServer() {
	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/json", handleJSON)

	httpSrv := httplog.MakeHTTPServer(mux)
	httpSrv.Addr = ":8100"

	fmt.Println(httpSrv.Addr)
	httpSrv.ListenAndServe()

	// Output: :8100
}

// simplest possible server that returns url as plain text
func handleIndex(w http.ResponseWriter, r *http.Request) {
	//msg := fmt.Sprintf("You've called url %s", r.URL.String())
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK) // 200
	bytes, _ := ioutil.ReadAll(r.Body)
	w.Write(bytes)
}

func handleJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK) // 200
	w.Write([]byte(`{"name":"bingoohuang"}`))
}
