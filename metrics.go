package httplog

import (
	"net/http"
	"net/http/httptest"
	"time"
)

const maxSize = 1000

// Metrics holds metrics captured from CaptureMetrics.
type Metrics struct {
	// Code is the first http response code passed to the WriteHeader func of
	// the ResponseWriter. If no such call is made, a default code of 200 is
	// assumed instead.
	Code  int
	Start time.Time
	End   time.Time
	// Duration is the time it took to execute the handler.
	Duration time.Duration
	// Written is the number of bytes successfully written by the Write or
	// ReadFrom function of the ResponseWriter. ResponseWriters may also write
	// data to their underlying connection directly (e.g. headers), but those
	// are not tracked. Therefore the number of Written bytes will usually match
	// the size of the response body.
	Written  int64
	RespBody string
}

// CaptureMetrics wraps the given hnd, executes it with the given w and r, and
// returns the metrics it captured from it.
func CaptureMetrics(hnd http.Handler, w http.ResponseWriter, r *http.Request) Metrics {
	return CaptureMetricsFn(w, func(ww http.ResponseWriter) {
		hnd.ServeHTTP(ww, r)
	})
}

// CaptureMetricsFn wraps w and calls fn with the wrapped w and returns the
// resulting metrics. This is very similar to CaptureMetrics (which is just
// sugar on top of this func), but is a more usable interface if your
// application doesn't use the Go http.Handler interface.
func CaptureMetricsFn(w http.ResponseWriter, fn func(http.ResponseWriter)) Metrics {
	m := Metrics{Start: time.Now()}
	rec := httptest.NewRecorder()

	fn(rec)

	m.Duration = time.Since(m.Start)
	m.End = time.Now()
	m.Code = rec.Code

	if rec.Body.Len() <= maxSize {
		m.RespBody = rec.Body.String()
	} else {
		m.RespBody = string(rec.Body.Bytes()[:maxSize]) + "..."
	}

	for k, v := range rec.Header() {
		w.Header()[k] = v
	}

	w.WriteHeader(rec.Code)
	m.Written, _ = rec.Body.WriteTo(w)

	return m
}
