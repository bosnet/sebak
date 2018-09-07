package httputils

import "net/http"

// IsEventStream checks request header accept is text/event-stream
func IsEventStream(r *http.Request) bool {
	if r.Header.Get("Accept") == "text/event-stream" {
		return true

	}
	return false
}
