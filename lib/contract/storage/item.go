package storage

import (
	common "boscoin.io/sebak/lib/common"
)

type Item struct {
	Value []byte
}

func (i *Item) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(i)
	return
}
func (i *Item) Deserialize(encoded []byte) (err error) {
	return common.DecodeJSONValue(encoded, i)
}
