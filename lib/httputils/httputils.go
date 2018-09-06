package httputils

import "net/http"

func IsEventStream(r *http.Request) bool {
	if r.Header.Get("Accept") == "text/event-stream" {
		return true

	}
	return false
}
