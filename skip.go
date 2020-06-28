package httplog

import (
	"strings"
)

func skipLoggingBefore(ri *Req, option *Option) bool {
	switch {
	case IsWsRequest(ri.URL):
		return true
	case option.Ignore:
		return true
	case ri.URL == "/favicon.png" || ri.URL == "/favicon.ico":
		return true
	case strings.HasSuffix(ri.URL, ".css"):
		return true
	}

	return false
}
