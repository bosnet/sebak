package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/store/statestore"
	"boscoin.io/sebak/lib/contract"
)

type OperationBodyContractDeploy struct {
	Target string `json:"target"`
	Amount Amount `json:"amount"`
	CodeType int `json:"codeType"`
	Code   string `json:"code"`
}

func NewOperationBodyContractDeploy(target string, amount Amount, codeType int, code string) OperationBodyContractDeploy {
	return OperationBodyContractDeploy{
		Target: target,
		Amount: amount,
		CodeType: codeType,
		Code: code,
	}
}

func (o OperationBodyContractDeploy) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyContractDeploy) IsWellFormed([]byte) (err error) {
	if _, err = keypair.Parse(o.Target); err != nil {
		return
	}

	if int64(o.Amount) < 1 {
		err = fmt.Errorf("invalid `Amount`")
		return
	}

	if len(o.Code) == 0 {
		err = fmt.Errorf("invalid `Code`")
	}

	return
}

func (o OperationBodyContractDeploy) Validate(st sebakstorage.LevelDBBackend) (err error) {
	return
}

func (o OperationBodyContractDeploy) TargetAddress() string {
	return o.Target
}

func (o OperationBodyContractDeploy) GetAmount() Amount {
	return o.Amount
}

func (o OperationBodyContractDeploy) Do(st *sebakstorage.LevelDBBackend, tx Transaction) (err error) {
	var baSource *BlockAccount
	if baSource, err = GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	stateStore := statestore.NewStateStore(st)
	stateClone := statestore.NewStateClone(stateStore)
	ctx := &context.Context{
		SenderAccount: baSource,
		StateStore:    stateStore,
		StateClone:    stateClone,
	}


	err = contract.Deploy(ctx, payload.CodeType(o.CodeType), []byte(o.Code)) //TODO: Where to pass the return value?
	return
}

func FinishOperationBodyContractDeploy (st *sebakstorage.LevelDBBackend, tx Transaction, op Operation) (err error) {
	var baSource *BlockAccount
	if baSource, err = GetBlockAccount(st, tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}
	stateStore := statestore.NewStateStore(st)
	stateClone := statestore.NewStateClone(stateStore)
	ctx := &context.Context{
		SenderAccount: baSource,
		StateStore:    stateStore,
		StateClone:    stateClone,
	}


	err = contract.Deploy(ctx, payload.CodeType(op.B.(OperationBodyContractDeploy).CodeType), []byte(op.B.(OperationBodyContractDeploy).Code)) //TODO: Where to pass the return value?
	return
}