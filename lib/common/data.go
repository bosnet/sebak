package common

import "encoding/json"

func EncodeJSONValue(v interface{}) (b []byte, err error) {
	return json.Marshal(v)
}

func DecodeJSONValue(b []byte, v interface{}) (err error) {
	if err = json.Unmarshal(b, v); err != nil {
		return
	}
	return
}

type Serializable interface {
	Serialize() ([]byte, error)
}
