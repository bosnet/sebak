package transaction

import (
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type OperationBodyCreateAccount struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
	Linked string        `json:"linked,omitempty"`
}

func NewOperationBodyCreateAccount(target string, amount common.Amount, linked string) OperationBodyCreateAccount {
	return OperationBodyCreateAccount{
		Target: target,
		Amount: amount,
		Linked: linked,
	}
}

// Implement transaction/operation : OperationBody.IsWellFormed
func (o OperationBodyCreateAccount) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.ErrorOperationAmountUnderflow
		return
	}

	if o.Amount < common.BaseReserve {
		err = errors.ErrorInsufficientAmountNewAccount
		return
	}

	return
}

func (o OperationBodyCreateAccount) TargetAddress() string {
	return o.Target
}

func (o OperationBodyCreateAccount) GetAmount() common.Amount {
	return o.Amount
}
