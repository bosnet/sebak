package sebak

import (
	"errors"
	"fmt"

	"boscoin.io/sebak/lib/contract/jsvm"
	"boscoin.io/sebak/lib/contract/native"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/contract/wasm"
)

type ContractExecutor interface {
	Execute(*payload.ExecCode) (*value.Value, error)
}

func NewContractExecutor(ctx *ContractContext, execCode *payload.ExecCode) (ContractExecutor, error) {
	var ex ContractExecutor
	contractAddress := execCode.ContractAddress
	api := NewContractAPI(ctx, contractAddress)

	if native.HasContract(contractAddress) {
		ex = native.NewNativeExecutor(ctx, api)
	} else {
		deployCode, err := ctx.StateStore.GetDeployCode(execCode.ContractAddress)
		if err != nil {
			return nil, err
		}
		switch deployCode.Type {
		case payload.JavaScript:
			ex = jsvm.NewOttoExecutor(ctx, api, deployCode)
		case payload.WASM:
			ex = wasm.NewWasmExecutor(ctx)
		default:
			return nil, errors.New("not supported language")
		}
	}
	return ex, nil
}

func ContractExecute(ctx *ContractContext, execCode *payload.ExecCode) (*value.Value, error) {
	ex, err := NewContractExecutor(ctx, execCode)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	if ex == nil {
		return nil, fmt.Errorf("not found")
	}
	ret, err := ex.Execute(execCode)
	return ret, err
}
