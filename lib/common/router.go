package common

import (
	"net/http"

	"github.com/gorilla/mux"
	"strings"
)

func PostAndJSONMatcher(r *http.Request, rm *mux.RouteMatch) bool {
	if r.Method == "POST" {
		if r.Header.Get("Content-Type") != "application/json" {
			return false
		}
	}

	return true
}


func PostAndJSONMatcherForClient(r *http.Request, rm *mux.RouteMatch) bool {
	if r.Method == "POST" {
		ct := r.Header.Get("Content-Type")
		if len(strings.TrimSpace(ct)) < 1 {
			return false
		}
		spl := strings.SplitN(ct, ";", 2)
		if len(spl) == 2 {
			ct = spl[0]
		}

		if strings.TrimSpace(ct) != "application/json" {
			return false
		}
	}

	return true
}
