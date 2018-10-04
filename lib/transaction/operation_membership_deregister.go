package transaction

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
)

type OperationBodyMembershipDeregister struct {
	Target string `json:"target"`
}

func NewOperationBodyMembershipDeregister(target string, amount common.Amount) OperationBodyMembershipDeregister {
	return OperationBodyMembershipDeregister{
		Target: target,
	}
}

func (o OperationBodyMembershipDeregister) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}
func (o OperationBodyMembershipDeregister) IsWellFormed([]byte) (err error) {
	// TODO : validate sender account signature
	return
}
