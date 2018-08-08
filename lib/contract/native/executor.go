package native

import (
	"fmt"

	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
)

type ExecFunc func(e *NativeExecutor, code *payload.ExecCode) (*value.Value, error)

type NativeExecutor struct {
	Context *context.Context
	api     *api.API

	execFuncs map[string]ExecFunc
}

func NewNativeExecutor(ctx *context.Context, api *api.API) *NativeExecutor {
	ex := &NativeExecutor{
		Context:   ctx,
		api:       api,
		execFuncs: map[string]ExecFunc{},
	}

	return ex
}

func (ex *NativeExecutor) Execute(c *payload.ExecCode) (*value.Value, error) {
	//TODO(anarcher)
	ex.loadFuncs(c.ContractAddress)

	if f, ok := ex.execFuncs[c.Method]; ok {
		returnCode, err := f(ex, c)
		return returnCode, err
	}

	//TODO(anarcher) define error
	return nil, fmt.Errorf("not found")
}

func (ex *NativeExecutor) RegisterFunc(name string, f ExecFunc) {
	ex.execFuncs[name] = f
}

func (ex *NativeExecutor) loadFuncs(addr string) {
	if r, ok := contracts[addr]; ok {
		r(ex)
	}
}
