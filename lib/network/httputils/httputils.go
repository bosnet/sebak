package httputils

import (
	"net/http"

	"boscoin.io/sebak/lib/error"
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
