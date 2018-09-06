package httputils

import (
	"encoding/json"
	"net/http"

	common "boscoin.io/sebak/lib/common"
)

func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)

	if s, ok := v.(common.Serializable); ok {
		return writeSerializable(w, s)
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

func writeSerializable(w http.ResponseWriter, s common.Serializable) error {

	bs, err := s.Serialize()
	if err != nil {
		return err
	}

	if _, err := w.Write(bs); err != nil {
		return err
	}

	return nil
}
