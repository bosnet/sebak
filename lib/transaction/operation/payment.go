package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

type Payment struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
}

func NewPayment(target string, amount common.Amount) Payment {
	return Payment{
		Target: target,
		Amount: amount,
	}
}

func (o Payment) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

// Implement transaction/operation : IsWellFormed
func (o Payment) IsWellFormed(common.Config) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.OperationAmountUnderflow
		return
	}

	return
}

func (o Payment) TargetAddress() string {
	return o.Target
}

func (o Payment) GetAmount() common.Amount {
	return o.Amount
}

// If isSourceLinked is true, tx.Source is frozen account.
func (o Payment) HasFee(isSourceLinked bool) bool {
	if isSourceLinked {
		return false
	}
	return true
}
