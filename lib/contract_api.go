package sebak

import (
	"time"

	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/contract/value"
)

//TODO: For prototype testing
func (a *ContractAPI) Helloworld(greeting string) (string, error) {
	return greeting + " WORLD!!", nil
}

// Contract API is used to executors
type ContractAPI struct {
	contractAddress string // the current contract address
	ctx             *ContractContext
	stateClone      *StateClone
}

func NewContractAPI(ctx *ContractContext, contractAddr string) *ContractAPI {
	api := &ContractAPI{
		contractAddress: contractAddr,
		ctx:             ctx,
		stateClone:      ctx.StateClone,
	}
	return api
}

// Read a item of a key from this contract own storage
func (a *ContractAPI) GetStorageItem(key string) (*storage.StorageItem, error) {
	return a.stateClone.GetStorageItem(a.contractAddress, key)
}

// Write a item to this contract's own storage
func (a *ContractAPI) PutStorageItem(key string, item *storage.StorageItem) error {
	return a.stateClone.PutStorageItem(a.contractAddress, key, item)
}

// Get this contract's balance
func (a *ContractAPI) GetBalance() (string, error) {
	ba, err := a.stateClone.GetAccount(a.contractAddress)
	if err != nil {
		return "0", err
	}
	return ba.Balance, nil
}

// Call another contract
func (a *ContractAPI) CallContract(execCode *payload.ExecCode) (v *value.Value, err error) {
	v, err = ContractExecute(a.ctx, execCode)
	return
}

// Return block height
func (a *ContractAPI) GetBlockHeight() int64 {
	//TODO(anarcher):
	return 0
}

func (a *ContractAPI) Now() time.Time {
	return time.Now().UTC()
}
