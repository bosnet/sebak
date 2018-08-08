package api

import (
	"time"

	sebak "boscoin.io/sebak/lib"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/store/statestore"
)

type CallContract func(ctx *context.Context, execCode *payload.ExecCode) (*value.Value, error)

//TODO: For prototype testing
func (a *API) Helloworld(greeting string) (string, error) {
	return greeting + " WORLD!!", nil
}

// Contract API is used to executors
type API struct {
	contractAddress string // the current contract address
	ctx             *context.Context
	stateClone      *statestore.StateClone
	callContract    CallContract
}

func NewAPI(ctx *context.Context, contractAddr string, callc CallContract) *API {
	api := &API{
		contractAddress: contractAddr,
		ctx:             ctx,
		stateClone:      ctx.StateClone,
		callContract:    callc,
	}
	return api
}

// Read a item of a key from this contract own storage
func (a *API) GetStorageItem(key string) (*statestore.StorageItem, error) {
	return a.stateClone.GetStorageItem(a.contractAddress, key)
}

// Write a item to this contract's own storage
func (a *API) PutStorageItem(key string, item *statestore.StorageItem) error {
	return a.stateClone.PutStorageItem(a.contractAddress, key, item)
}

// Get this contract's balance
func (a *API) GetBalance() (sebak.Amount, error) {
	ba, err := a.stateClone.GetAccount(a.contractAddress)
	if err != nil {
		return 0, err
	}
	balance, err := sebak.AmountFromString(ba.Balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

// Call another contract
func (a *API) CallContract(execCode *payload.ExecCode) (v *value.Value, err error) {
	v, err = a.callContract(a.ctx, execCode)
	return
}

// Return block height
func (a *API) GetBlockHeight() int64 {
	//TODO(anarcher):
	return 0
}

func (a *API) Now() time.Time {
	return time.Now().UTC()
}
