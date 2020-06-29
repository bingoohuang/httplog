package httplog

import (
	"strings"
)

func (l *Log) skipLoggingBefore(mux *Mux) bool {
	switch {
	case l.Biz == "Noname" && mux.muxOption.IgnoreBizNoname:
		return true
	case IsWsRequest(l.URL):
		return true
	case l.Option.Ignore:
		return true
	case l.URL == "/favicon.png" || l.URL == "/favicon.ico":
		return true
	case strings.HasSuffix(l.URL, ".css"):
		return true
	}

	return false
}
