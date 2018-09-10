package transaction

import (
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
)

type OperationBodyCreateAccount struct {
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
		err = fmt.Errorf("invalid `Amount`: lower than 1")
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

