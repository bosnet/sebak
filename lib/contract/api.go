package contract

import (
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/storage"
)

//TODO: For prototype testing
func (a *API) Helloworld(greeting string) (string, error) {
	return greeting + " WORLD!!", nil
}

// Contract API is used to executors
type API struct {
	contractAddress string // the current contract address
	ctx             *Context
	stateDB         sebakstorage.DBBackend
}

func NewAPI(ctx *Context, contractAddr string) *API {
	api := &API{
		contractAddress: contractAddr,
		ctx:             ctx,
		stateDB:         ctx.db,
	}
	return api
}

// Read a item of a key from this contract own storage
func (a *API) GetStorageItem(key string) (*storage.StorageItem, error) {
	return storage.GetStorageItem(a.stateDB, a.contractAddress, key)
}

// Write a item to this contract's own storage
func (a *API) PutStorageItem(key string, item *storage.StorageItem) error {
	item.Address = a.contractAddress
	item.Key = key
	return item.Save(a.stateDB)
}

// Get this contract's balance
func (a *API) GetBalance() (string, error) {
	ba, err := block.GetBlockAccount(a.stateDB, a.contractAddress)
	if err != nil {
		return "0", err
	}
	return ba.Balance, nil
}

// Call another contract
func (a *API) CallContract(execCode *payload.ExecCode) (v *value.Value, err error) {
	v, err = Execute(a.ctx, execCode)
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
