package httputils

import (
	"encoding/json"
	"net/http"

	"github.com/nvellon/hal"

	"boscoin.io/sebak/lib/errors"
)

type HALResource interface {
	Resource() *hal.Resource
}

// MustWriteJSON writes the value or an error of it to the http response as json
func MustWriteJSON(w http.ResponseWriter, code int, v interface{}) {
	if err := WriteJSON(w, code, v); err != nil {
		WriteJSONError(w, err)
	}
}

// WriteJSONError writes the error to the http response as json encoding
func WriteJSONError(w http.ResponseWriter, err error) {
	code := StatusCode(err) //TODO(anarcher): ErrorStateCode is more suitable?
	if sebakError, ok := err.(*errors.Error); ok {
		if status := sebakError.GetData("status"); status != nil {
			code = status.(int)
		}

	}

	if err := WriteJSON(w, code, err); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// WriteJSON writes the value v to the http response as json encoding
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	if _, ok := v.(HALResource); ok {
		w.Header().Set("Content-Type", "application/hal+json")
	} else if e, ok := v.(error); ok {
		w.Header().Set("Content-Type", "application/problem+json")
		v = NewErrorProblem(e, code)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(code)

	var bs []byte
	var err error
	if marshaler, ok := v.(json.Marshaler); ok {
		bs, err = marshaler.MarshalJSON()
	} else {
		bs, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}

	return nil
}
