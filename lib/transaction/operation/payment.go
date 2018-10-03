package operation

import (
	"encoding/json"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
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
func (o Payment) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.ErrorOperationAmountUnderflow
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
