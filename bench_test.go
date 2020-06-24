package httplog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bingoohuang/httplog"
)

func BenchmarkBaseline(b *testing.B) {
	benchmark(b, false)
}

func BenchmarkCaptureMetrics(b *testing.B) {
	benchmark(b, true)
}

func benchmark(b *testing.B, captureMetrics bool) {
	b.StopTimer()
	dummyH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := dummyH
	if captureMetrics {
		h = func(w http.ResponseWriter, r *http.Request) {
			httplog.CaptureMetrics(dummyH, w, r)
		}
	}
	s := httptest.NewServer(h)
	defer s.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := http.Get(s.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}
