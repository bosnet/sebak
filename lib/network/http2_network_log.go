package network

import (
	"net/http"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
)

type HTTP2ErrorLog15Writer struct {
	l logging.Logger
}

func (w HTTP2ErrorLog15Writer) Write(b []byte) (int, error) {
	w.l.Error("error", "error", string(b))
	return 0, nil
}

type HTTP2ResponseLog15Writer struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *HTTP2ResponseLog15Writer) Header() http.Header {
	return l.w.Header()
}

func (l *HTTP2ResponseLog15Writer) Write(b []byte) (int, error) {
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *HTTP2ResponseLog15Writer) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *HTTP2ResponseLog15Writer) Status() int {
	return l.status
}

func (l *HTTP2ResponseLog15Writer) Size() int {
	return l.size
}

func (l *HTTP2ResponseLog15Writer) Flush() {
	f, ok := l.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

type HTTP2Log15Handler struct {
	log     logging.Logger
	handler http.Handler
}

var HeaderKeyFiltered []string = []string{
	"Content-Length",
	"Content-Type",
	"Accept",
	"Accept-Encoding",
	"User-Agent",
}

// ServeHTTP will log in 2 phase, when request received and response sent. This
// was derived from github.com/gorilla/handlers/handlers.go
func (l HTTP2Log15Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid := common.GenerateUUID()

	uri := r.RequestURI
	if r.ProtoMajor == 2 && r.Method == "CONNECT" {
		uri = r.Host
	}
	if uri == "" {
		uri = r.URL.RequestURI()
	}

	header := http.Header{}
	for key, value := range r.Header {
		if _, found := common.InStringArray(HeaderKeyFiltered, key); found {
			continue
		}
		header[key] = value
	}

	l.log.Debug(
		"request",
		"content-length", r.ContentLength,
		"content-type", r.Header.Get("Content-Type"),
		"headers", header,
		"host", r.Host,
		"id", uid,
		"method", r.Method,
		"proto", r.Proto,
		"referer", r.Referer(),
		"remote", r.RemoteAddr,
		"uri", uri,
		"user-agent", r.UserAgent(),
	)

	writer := &HTTP2ResponseLog15Writer{w: w}
	l.handler.ServeHTTP(writer, r)

	l.log.Debug(
		"response",
		"id", uid,
		"status", writer.Status(),
		"size", writer.Size(),
	)
}
