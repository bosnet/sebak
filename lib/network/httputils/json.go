package httputils

import (
	"encoding/json"
	"github.com/nvellon/hal"
	"net/http"
)

type HALResource interface {
	Resource() *hal.Resource
}

// WriteJSON writes the value v to the http response as json encoding
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	if h, ok := v.(HALResource); ok {
		w.Header().Set("Content-Type", "application/hal+json")
		v = h.Resource()
	} else if e, ok := v.(error); ok {
		w.Header().Set("Content-Type", "application/json")
		v = NewErrorProblem(e, code)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(code)

	bs, err := json.Marshal(v)
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}

	return nil
}
