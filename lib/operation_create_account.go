package sebak

import (
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyCreateAccount struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

func NewOperationBodyCreateAccount(target string, amount Amount) OperationBodyCreateAccount {
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

func (o OperationBodyCreateAccount) Validate(st sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Target` is not in `Block Account`

	return
}

func (o OperationBodyCreateAccount) TargetAddress() string {
	return o.Target
}

func (o OperationBodyCreateAccount) GetAmount() Amount {
	return o.Amount
}

func FinishOperationCreateAccount(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *BlockAccount
	if baSource, err = GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = GetBlockAccount(st, op.B.TargetAddress()); err == nil {
		err = sebakerror.ErrorBlockAccountAlreadyExists
		return
	} else {
		err = nil
	}

	baTarget = NewBlockAccount(
		op.B.TargetAddress(),
		op.B.GetAmount(),
		tx.NextTargetCheckpoint(),
	)
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("new account created", "source", baSource, "target", baTarget)

	return
}
