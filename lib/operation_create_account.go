package sebak

import (
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
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

func (o OperationBodyCreateAccount) Validate(st *storage.LevelDBBackend) (err error) {
	var exists bool
	if exists, err = block.ExistBlockAccount(st, o.Target); err == nil && exists {
		err = errors.ErrorBlockAccountAlreadyExists
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

func FinishOperationCreateAccount(st *storage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err == nil {
		err = errors.ErrorBlockAccountAlreadyExists
		return
	} else {
		err = nil
	}

	baTarget = block.NewBlockAccount(
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
