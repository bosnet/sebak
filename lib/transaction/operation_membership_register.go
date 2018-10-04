package transaction

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
)

type OperationBodyMembershipRegister struct {
	Target string `json:"target"`
}

func NewOperationBodyMembershipRegister(target string, amount common.Amount) OperationBodyMembershipRegister {
	return OperationBodyMembershipRegister{
		Target: target,
	}
}

func (o OperationBodyMembershipRegister) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}
func (o OperationBodyMembershipRegister) IsWellFormed([]byte) (err error) {
	return
}
