package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
	sebakcommon "boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyContractExecute struct {
	Target string             `json:"target"`
	Amount sebakcommon.Amount `json:"amount"`
	Method string             `json:"method"`
	Args   []string           `json:"args"`
}

func NewOperationBodyContractExecute(target string, amount sebakcommon.Amount, method string, args []string) OperationBodyContractExecute {
	return OperationBodyContractExecute{
		Target: target,
		Amount: amount,
		Method: method,
		Args:   args,
	}
}

func (o OperationBodyContractExecute) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyContractExecute) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	if len(o.Method) == 0 {
		err = fmt.Errorf("invalid `Method`")
	}

	return
}

func (o OperationBodyContractExecute) Validate(st sebakstorage.LevelDBBackend) (err error) {
	return
}

func (o OperationBodyContractExecute) TargetAddress() string {
	return o.Target
}

func (o OperationBodyContractExecute) GetAmount() sebakcommon.Amount {
	return o.Amount
}

func FinishOperationBodyContractExecute(st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}

	ctx := contract.NewContext(baSource, st) // st as statedb

	exCode := &payload.ExecCode{
		ContractAddress: baTarget.Address,
		Method:          op.B.(OperationBodyContractExecute).Method,
		Args:            op.B.(OperationBodyContractExecute).Args,
	}

	_, err = contract.Execute(ctx, exCode) //TODO: Where to pass the return value?
	return
}
