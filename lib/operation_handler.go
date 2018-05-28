package sebak

import (
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
)

// FinishOperation do finish the task after consensus by the type of each operation.
func FinishOperation(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	switch op.H.Type {
	case OperationCreateAccount:
		return FinishOperationCreateAccount(st, tx, op)
	case OperationPayment:
		return FinishOperationPayment(st, tx, op)
	default:
		err = sebakerror.ErrorUnknownOperationType
		return
	}

	return
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

func FinishOperationPayment(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	return
}
