package sebak

import (
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
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

func (o OperationBodyCreateAccount) IsWellFormed() (err error) {
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

	// TODO make initial checkpoint for newly created account
	hashed := sebakcommon.MustMakeObjectHash("")
	checkpoint := base58.Encode(hashed)

	baTarget = NewBlockAccount(
		op.B.TargetAddress(),
		op.B.GetAmount().String(),
		checkpoint,
	)
	if err = baTarget.Save(st); err != nil {
		return
	}

	// TODO make new checkpoint
	var expected Amount
	if expected, err = baSource.GetBalanceAmount().Add(int64(op.B.GetAmount()) * -1); err != nil {
		return
	}

	baSource.EnsureUpdate(
		int64(op.B.GetAmount())*-1,
		checkpoint,
		int64(expected),
	)
	if err = baSource.Save(st); err != nil {
		return
	}

	return
}
