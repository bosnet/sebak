package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyPayment struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

func NewOperationBodyPayment(target string, amount Amount) OperationBodyPayment {
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

func (o OperationBodyPayment) Validate(st sebakstorage.LevelDBBackend) (err error) {
	// TODO check whether `Target` is in `Block Account`
	// TODO check over minimum balance
	return
}

func (o OperationBodyPayment) TargetAddress() string {
	return o.Target
}

func (o OperationBodyPayment) GetAmount() Amount {
	return o.Amount
}

func FinishOperationPayment(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *BlockAccount
	if baSource, err = GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = GetBlockAccount(st, op.B.TargetAddress()); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}

	if err = baTarget.Deposit(op.B.GetAmount(), tx.NextCheckpoint()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("payment done", "source", baSource, "target", baTarget, "amount", op.B.GetAmount())

	return
}
