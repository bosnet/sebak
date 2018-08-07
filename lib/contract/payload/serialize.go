package payload

import "encoding/json"

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

func EncodeJSONValue(v interface{}) (b []byte, err error) {
	if b, err = json.Marshal(v); err != nil {
		return
	}

	return
}

func DecodeJSONValue(b []byte, v interface{}) (err error) {
	if err = json.Unmarshal(b, v); err != nil {
		return
	}
	return
}
