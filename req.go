package httplog

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Log describes info about HTTP request.
type Log struct {
	HandlerName string

	// Method is GET etc.
	Method string
	URL    string
	IPAddr string

	RespHeader http.Header
	ReqBody    string

	// RespCode, like 200, 404.
	RespCode int
	// ReqHeader records the response header.
	ReqHeader http.Header
	// RespSize is number of bytes of the response sent.
	RespSize int64
	// RespBody is the response body(limit to 1000).
	RespBody string

	// Start records the start time of the request.
	Start time.Time
	// End records the end time of the request.
	End time.Time
	// Duration means how long did it take to.
	Duration time.Duration
	Attrs    Attrs
}

// Store defines the interface to Store a log.
type Store interface {
	// Store stores the log in database like MySQL, InfluxDB, and etc.
	Store(log *Log)
}

// LogrusStore stores the log as logurs info.
type LogrusStore struct{}

// Store stores the log in database like MySQL, InfluxDB, and etc.
func (s *LogrusStore) Store(log *Log) {
	logrus.Infof("http:%+v\n", log)
}