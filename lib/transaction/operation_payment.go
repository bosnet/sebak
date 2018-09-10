package transaction

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyPayment struct {
	Target string        `json:"target"`
	Amount common.Amount `json:"amount"`
}

func NewOperationBodyPayment(target string, amount common.Amount) OperationBodyPayment {
	return OperationBodyPayment{
		Target: target,
		Amount: amount,
	}
}

func (o OperationBodyPayment) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyPayment) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	return
}

func (o OperationBodyPayment) Validate(st *storage.LevelDBBackend) (err error) {
	var exists bool
	if exists, err = block.ExistBlockAccount(st, o.Target); err == nil && !exists {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	return
}

func (o OperationBodyPayment) TargetAddress() string {
	return o.Target
}

func (o OperationBodyPayment) GetAmount() common.Amount {
	return o.Amount
}

func FinishOperationPayment(st *storage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	if err = baTarget.Deposit(op.B.GetAmount()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("payment done", "source", baSource, "target", baTarget, "amount", op.B.GetAmount())

	return
}
