package httputils

import (
	"net/http"

	"boscoin.io/sebak/lib/errors"
	"io"
)

// IsEventStream checks request header accept is text/event-stream
func IsEventStream(r *http.Request) bool {
	if r.Header.Get("Accept") == "text/event-stream" {
		return true

	}
	return false
}

var (
	// ErrorsToStatus defines errors.Error does not have 400 status code.
	ErrorsToStatus = map[uint]int{
		errors.ErrorTooManyRequests.Code:               http.StatusTooManyRequests,
		errors.ErrorBlockTransactionDoesNotExists.Code: http.StatusNotFound,
		errors.ErrorBlockAccountDoesNotExists.Code:     http.StatusNotFound,
	}
)

func StatusCode(err error) int {
	if e, ok := err.(*errors.Error); ok {
		if c, ok := ErrorsToStatus[e.Code]; ok {
			return c
		}
		return 400
	}

	return 500
}

// Create our own ResponseWriterInterceptor to wrap a standard http.ResponseWriter
// so we can store the status code.
type ResponseWriterInterceptor struct {
	http.ResponseWriter
	code   int
	writer io.Writer
}

func NewResponseWriterInterceptor(w http.ResponseWriter, writer io.Writer) *ResponseWriterInterceptor {
	// Default the status code to 200
	return &ResponseWriterInterceptor{ResponseWriter: w, code: 200, writer: writer}
}

// Satisfy the http.ResponseWriter interface
func (w ResponseWriterInterceptor) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *ResponseWriterInterceptor) WriteHeader(statusCode int) {
	w.code = statusCode
}

func (w ResponseWriterInterceptor) Write(data []byte) (int, error) {

	return w.writer.Write(data)
}

func (w ResponseWriterInterceptor) StatusCode() int {
	return w.code
}

func (w ResponseWriterInterceptor) WriteToOrigin(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

func (w ResponseWriterInterceptor) WriteHeaderToOrigin(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}
