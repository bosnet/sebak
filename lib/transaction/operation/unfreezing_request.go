package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
)

type UnfreezeRequest struct{}

func NewUnfreezeRequest() UnfreezeRequest {
	return UnfreezeRequest{}
}

func (o UnfreezeRequest) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o UnfreezeRequest) IsWellFormed([]byte, common.Config) (err error) {
	return
}
