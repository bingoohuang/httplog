package httplog_test

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bingoohuang/httplog"
)

// nolint:funlen
func TestCaptureMetrics(t *testing.T) {
	// Some of the edge cases tested below cause the net/http pkg to log some
	// messages that add a lot of noise to the `go tc -v` output, so we discard
	// the log here.
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	tests := []struct {
		Handler      http.Handler
		WantDuration time.Duration
		WantWritten  int64
		WantCode     int
		WantErr      string
	}{
		{
			Handler:  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			WantCode: http.StatusOK,
		},
		{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("foo"))
				_, _ = w.Write([]byte("bar"))
				time.Sleep(25 * time.Millisecond)
			}),
			WantCode:     http.StatusBadRequest,
			WantWritten:  6,
			WantDuration: 25 * time.Millisecond,
		},
		{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("foo"))
				w.WriteHeader(http.StatusNotFound)
			}),
			WantCode: http.StatusOK,
		},
		{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("oh no")
			}),
			WantErr: "EOF",
		},
	}

	for i, tc := range tests {
		tc := tc

		func() {
			ch := make(chan httplog.Metrics, 1)
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ch <- httplog.CaptureMetrics(tc.Handler, w, r)
			})
			s := httptest.NewServer(h)

			defer s.Close()

			res, err := http.Get(s.URL)

			if !errContains(err, tc.WantErr) {
				t.Errorf("tc %d: got=%s want=%s", i, err, tc.WantErr)
			}

			if err != nil {
				return
			}

			defer res.Body.Close()

			m := <-ch

			switch {
			case m.Code != tc.WantCode:
				t.Errorf("tc %d: got=%d want=%d", i, m.Code, tc.WantCode)
			case m.Duration < tc.WantDuration:
				t.Errorf("tc %d: got=%s want=%s", i, m.Duration, tc.WantDuration)
			case m.Written < tc.WantWritten:
				t.Errorf("tc %d: got=%d want=%d", i, m.Written, tc.WantWritten)
			}
		}()
	}
}

func errContains(err error, s string) bool {
	var errS string
	if err == nil {
		errS = ""
	} else {
		errS = err.Error()
	}

	return strings.Contains(errS, s)
}
