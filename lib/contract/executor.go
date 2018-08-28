package contract

import (
	"errors"
	"fmt"

	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	_ "boscoin.io/sebak/lib/contract/init"
	"boscoin.io/sebak/lib/contract/jsvm"
	"boscoin.io/sebak/lib/contract/native"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/contract/wasm"
)

type Executor interface {
	Execute(*payload.ExecCode) (*value.Value, error)
}

func NewExecutor(ctx *context.Context, execCode *payload.ExecCode) (Executor, error) {
	var ex Executor
	contractAddress := execCode.ContractAddress
	api := api.NewAPI(ctx, contractAddress)

	if native.HasContract(contractAddress) {
		ex = native.NewNativeExecutor(ctx, api)
	} else {
		deployCode, err := ctx.GetDeployCode(execCode.ContractAddress)
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

func Execute(ctx *context.Context, execCode *payload.ExecCode) (*value.Value, error) {
	ex, err := NewExecutor(ctx, execCode)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	if ex == nil {
		return nil, fmt.Errorf("not found")
	}
	ret, err := ex.Execute(execCode)
	return ret, err
}
