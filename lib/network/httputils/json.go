package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/nvellon/hal"
)

type HALResource interface {
	Resource() *hal.Resource
}

// WriteJSONError writes the error to the http response as json encoding
func WriteJSONError(w http.ResponseWriter, err error) {
	code := StatusCode(err) //TODO(anarcher): ErrorStateCode is more suitable?

	if err := WriteJSON(w, code, err); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// WriteJSON writes the value v to the http response as json encoding
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	if h, ok := v.(HALResource); ok {
		w.Header().Set("Content-Type", "application/hal+json")
		v = h.Resource()
	} else if e, ok := v.(error); ok {
		w.Header().Set("Content-Type", "application/problem+json")
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
