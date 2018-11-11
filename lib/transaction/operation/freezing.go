package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

type Freezing struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
	Linked string        `json:"linked"`
}

func NewFreezing(target string, amount common.Amount, linked string) Freezing {
	return Freezing{
		Target: target,
		Amount: amount,
		Linked: linked,
	}
}

func (o Freezing) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

func (o Freezing) IsWellFormed(common.Config) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.OperationAmountUnderflow
		return
	}

	if o.Amount < common.BaseReserve {
		err = errors.InsufficientAmountNewAccount
		return
	}

	if o.Linked == "" {
		err = errors.InvalidLinkedValue
	}
	return
}

func (o Freezing) TargetAddress() string {
	return o.Target
}

func (o Freezing) GetAmount() common.Amount {
	return o.Amount
}

func (o Freezing) HasFee() bool {
	return false
}
