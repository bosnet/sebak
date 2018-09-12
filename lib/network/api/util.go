package api

import (
	"boscoin.io/sebak/lib/storage"
	"net/http"
	"strconv"
)

func parseQueryString(r *http.Request) *storage.IteratorOptions {

	var reverse bool
	var limit uint64
	var cursor []byte

	if v := r.FormValue("reverse"); v == "true" {
		reverse = true
	} else {
		reverse = false
	}

	if v := r.FormValue("limit"); v != "" {
		var err error
		var l int
		l, err = strconv.Atoi(v)
		if err != nil {
			limit = 0
		} else {
			limit = uint64(l)
		}
	} else {
		limit = 0
	}

	if limit > 1000 {
		limit = 1000
	}

	if v := r.FormValue("cursor"); v != "" {
		cursor = []byte(v)
	} else {
		cursor = nil
	}

	return &storage.IteratorOptions{
		Reverse: reverse,
		Limit:   limit,
		Cursor:  []byte(cursor),
	}
}
