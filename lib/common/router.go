package common

import (
	"net/http"

	"github.com/gorilla/mux"
)

func PostAndJSONMatcher(r *http.Request, rm *mux.RouteMatch) bool {
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "application/json" {
			return false
		}
	}

	return true
}
