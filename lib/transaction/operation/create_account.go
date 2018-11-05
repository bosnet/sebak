package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

type CreateAccount struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
	Linked string        `json:"linked,omitempty"`
}

func NewCreateAccount(target string, amount common.Amount, linked string) CreateAccount {
	return CreateAccount{
		Target: target,
		Amount: amount,
		Linked: linked,
	}
}

func (o CreateAccount) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}

// Implement transaction/operation : IsWellFormed
func (o CreateAccount) IsWellFormed(common.Config) (err error) {
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

	return
}

func (o CreateAccount) TargetAddress() string {
	return o.Target
}

func (o CreateAccount) GetAmount() common.Amount {
	return o.Amount
}
