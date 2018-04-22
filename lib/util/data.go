package util

import "encoding/json"

func EncodeJSONValue(v interface{}) (b []byte, err error) {
	if b, err = json.Marshal(v); err != nil {
		return
	}

	return
}
