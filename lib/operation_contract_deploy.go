package sebak

import (
	"encoding/json"
	"fmt"

	"github.com/stellar/go/keypair"

	sebakcommon "boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

type OperationBodyContractDeploy struct {
	Target   string             `json:"target"`
	Amount   sebakcommon.Amount `json:"amount"`
	CodeType int                `json:"codeType"`
	Code     string             `json:"code"`
}

func NewOperationBodyContractDeploy(target string, amount sebakcommon.Amount, codeType int, code string) OperationBodyContractDeploy {
	return OperationBodyContractDeploy{
		Target:   target,
		Amount:   amount,
		CodeType: codeType,
		Code:     code,
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

func (o OperationBodyContractDeploy) GetAmount() sebakcommon.Amount {
	return o.Amount
}

func FinishOperationBodyContractDeploy(st sebakstorage.DBBackend, tx Transaction, op Operation) (err error) {
	//TODO: use statedb
	/*
		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
			err = sebakerror.ErrorBlockAccountDoesNotExists
			return
		}

		ctx := context.NewContext(baSource, st)

		err = contract.Deploy(ctx, payload.CodeType(op.B.(OperationBodyContractDeploy).CodeType), []byte(op.B.(OperationBodyContractDeploy).Code)) //TODO: Where to pass the return value?
	*/
	return
}
