package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

type Unfreezing struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
}

func NewUnfreezing(target string, amount common.Amount) Unfreezing {
	return Unfreezing{
		Target: target,
		Amount: amount,
	}
}

func (o Unfreezing) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

// Implement transaction/operation : IsWellFormed
func (o Unfreezing) IsWellFormed(common.Config) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.OperationAmountUnderflow
		return
	}

	return
}

func (o Unfreezing) TargetAddress() string {
	return o.Target
}

func (o Unfreezing) GetAmount() common.Amount {
	return o.Amount
}

func (o Unfreezing) HasFee() bool {
	return false
}
