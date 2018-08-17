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
	Target string             `json:"target"`
	Amount sebakcommon.Amount `json:"amount"`
}

func NewOperationBodyCreateAccount(target string, amount sebakcommon.Amount) OperationBodyCreateAccount {
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

func (o OperationBodyCreateAccount) GetAmount() sebakcommon.Amount {
	return o.Amount
}

func FinishOperationCreateAccount(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err == nil {
		err = sebakerror.ErrorBlockAccountAlreadyExists
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
