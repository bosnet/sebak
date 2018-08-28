package api

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/value"
	"time"
)

//TODO: For prototype testing
func (a *API) Helloworld(greeting string) (string, error) {
	return greeting + " WORLD!!", nil
}

// Contract API is used to executors
type API struct {
	contractAddress string // the current contract address
	ctx             *context.Context
}

func NewAPI(ctx *context.Context, contractAddr string) *API {
	api := &API{
		contractAddress: contractAddr,
		ctx:             ctx,
	}
	return api
}

// Read a item of a key from this contract own storage
func (a *API) GetStorageItem(key []byte) (item *value.Value, err error) {
	hashKey := sebakcommon.BytesToHash(key)
	encoded := a.ctx.StateDB().GetState(a.contractAddress, hashKey)
	item = new(value.Value)
	item.Deserialize(encoded)

	return item, nil
}

// Write a item to this contract's own storage
func (a *API) PutStorageItem(key []byte, value *value.Value) (err error) {
	hashKey := sebakcommon.BytesToHash(key)
	encoded := value.Serialize()
	a.ctx.StateDB().SetState(a.contractAddress, hashKey, encoded)
	return
}

// Get this contract's balance
func (a *API) GetBalance() (balance *value.Value, err error) {
	balanceString := a.ctx.StateDB().GetBalance(a.contractAddress)
	amount := sebakcommon.MustAmountFromString(balanceString)
	balance, err = value.ToValue(uint64(amount))
	return balance, nil
}

// Call another contract
//func (a *API) CallContract(execCode *payload.ExecCode) (v *value.Value, err error) {
//	v, err = ExecuteContract(a.ctx, execCode)
//	return
//}

// Return block height
func (a *API) GetBlockHeight() int64 {
	//TODO(anarcher):
	return 0
}

func (a *API) Now() time.Time {
	return time.Now().UTC()
}
