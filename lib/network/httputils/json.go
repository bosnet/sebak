package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/nvellon/hal"
)

type HALResource interface {
	Resource() *hal.Resource
}

// WriteJSON writes the value v to the http response as json encoding
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.WriteHeader(code)

	if h, ok := v.(HALResource); ok {
		w.Header().Set("content-type", "application/hal+json")
		v = h.Resource()
	} else {
		w.Header().Set("content-type", "application/json")
	}

	bs, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}

	return nil
}
