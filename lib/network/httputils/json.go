package httputils

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes the value v to the http response as json encoding
func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("content-type", "application/json")
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
