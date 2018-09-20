package transaction

import (
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type OperationBodyCreateAccount struct {
	OperationBodyImpl
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
}

func NewOperationBodyCreateAccount(target string, amount common.Amount) OperationBodyCreateAccount {
	return OperationBodyCreateAccount{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyCreateAccount) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = errors.ErrorOperationAmountUnderflow
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
