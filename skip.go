package httplog

import (
	"strings"
)

func (ri *Log) skipLoggingBefore() bool {
	switch {
	case IsWsRequest(ri.URL):
		return true
	case ri.Option.Ignore:
		return true
	case ri.URL == "/favicon.png" || ri.URL == "/favicon.ico":
		return true
	case strings.HasSuffix(ri.URL, ".css"):
		return true
	}

	return false
}
