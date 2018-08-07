package api

import (
	sebak "boscoin.io/sebak/lib"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/store/statestore"
)

func (a *API) Helloworld(greeting string) (string, error) {
	return greeting + " WORLD!!", nil
}

// Contract API is used to executors
type API struct {
	//TODO
	ctx *context.Context
}

func NewAPI(ctx *context.Context) *API {
	api := &API{
		ctx: ctx,
	}
	return api
}

// Read a item of a key from this contract own storage
func (a *API) GetStorageItem(addr, key string) (*statestore.StorageItem, error) {

	return nil, nil
}

// Write a item to this contract's own storage
func (a *API) PutStorageItem(addr, key string, item *statestore.StorageItem) error {

	return nil
}

// Get this contract's balance
func (a *API) GetBalance() *sebak.Amount {
	return nil
}

// Call another contract
func (a *API) CallContract(addr string, execCode *payload.ExecCode) (*value.Value, error) {
	return nil, nil
}

// Return block height
func (a *API) GetBlockHeight() int64 {
	return 0
}
