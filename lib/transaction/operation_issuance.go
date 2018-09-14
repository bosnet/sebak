package transaction

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
)

type OperationBodyIssuance struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
}

func NewOperationBodyIssuance(target string, amount common.Amount) OperationBodyIssuance {
	return OperationBodyIssuance{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyIssuance) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyIssuance) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	return
}

func (o OperationBodyIssuance) TargetAddress() string {
	return o.Target
}

func (o OperationBodyIssuance) GetAmount() common.Amount {
	return o.Amount
}
