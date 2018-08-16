package sebakcommon

import "encoding/json"

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

type Serializable interface {
	Serialize() ([]byte, error)
}
