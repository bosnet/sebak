package types

import "gopkg.in/vmihailenco/msgpack.v3"

func Serialize(msg interface{}) ([]byte, error) {
	return msgpack.Marshal(msg)
}

func Deserialize(data []byte, msg interface{}) error {
	return msgpack.Unmarshal(data, msg)
}
