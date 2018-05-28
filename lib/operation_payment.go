package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
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

func (o OperationBodyPayment) IsWellFormed() (err error) {
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

	// TODO make initial checkpoint for newly created account
	hashed := sebakcommon.MustMakeObjectHash("")
	checkpoint := base58.Encode(hashed)

	// TODO make new checkpoint
	var expectedSource Amount
	if expectedSource, err = baSource.GetBalanceAmount().Add(int64(op.B.GetAmount()) * -1); err != nil {
		return
	}

	baSource.EnsureUpdate(
		int64(op.B.GetAmount())*-1,
		checkpoint,
		int64(expectedSource),
	)
	if err = baSource.Save(st); err != nil {
		return
	}

	var expectedTarget Amount
	if expectedTarget, err = baTarget.GetBalanceAmount().Add(int64(op.B.GetAmount())); err != nil {
		return
	}

	baTarget.EnsureUpdate(
		int64(op.B.GetAmount()),
		checkpoint,
		int64(expectedTarget),
	)
	if err = baTarget.Save(st); err != nil {
		return
	}

	return
}
