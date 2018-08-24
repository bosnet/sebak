package test

import "fmt"

/*
import (
	"fmt"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/contract/value"
)

type (
	HelloworldFunc     func(context.Context, string) (string, error)
	GetStorageItemFunc func(context.Context, string) (*storage.StorageItem, error)
	PutStorageItemFunc func(context.Context, string, *storage.StorageItem) error
	CallContractFunc   func(context.Context, *payload.ExecCode) (*value.Value, error)
)

type MockAPI struct {
	Address            string
	ctx                context.Context
	helloworldFunc     HelloworldFunc
	getStorageItemFunc GetStorageItemFunc
	putStorageItemFunc PutStorageItemFunc
	callContractFunc   CallContractFunc
}

func NewMockAPI(ctx context.Context, address string) *MockAPI {
	api := &MockAPI{
		Address: address,
		ctx:     ctx,
	}

	return api

}

func (a *MockAPI) SetFunc(f interface{}) (err error) {
	switch f.(type) {
	case HelloworldFunc:
		a.helloworldFunc = f.(HelloworldFunc)
	case GetStorageItemFunc:
		a.getStorageItemFunc = f.(GetStorageItemFunc)
	case PutStorageItemFunc:
		a.putStorageItemFunc = f.(PutStorageItemFunc)
	case CallContractFunc:
		a.callContractFunc = f.(CallContractFunc)
	default:
		err = fmt.Errorf("this func doesn't match for mock api")
	}
	return

}

func (a *MockAPI) SetHelloworldFunc(f HelloworldFunc) {
	a.helloworldFunc = f
}

func (a *MockAPI) SetGetStroageItemFunc(f GetStorageItemFunc) {
	a.getStorageItemFunc = f
}

func (a *MockAPI) SetPutStorageItemFunc(f PutStorageItemFunc) {
	a.putStorageItemFunc = f
}

func (a *MockAPI) SetCallContractfunc(f CallContractFunc) {
	a.callContractFunc = f
}

func (a *MockAPI) Helloworld(msg string) (string, error) {
	a.checkFunc(a.helloworldFunc)
	return a.helloworldFunc(a.ctx, msg)
}

func (a *MockAPI) GetStorageItem(key string) (*storage.StorageItem, error) {
	a.checkFunc(a.getStorageItemFunc)
	return a.getStorageItemFunc(a.ctx, key)
}

func (a *MockAPI) PutStorageItem(key string, item *storage.StorageItem) error {
	a.checkFunc(a.putStorageItemFunc)
	return a.putStorageItemFunc(a.ctx, key, item)
}

func (a *MockAPI) CallContract(code *payload.ExecCode) (*value.Value, error) {
	a.checkFunc(a.callContractFunc)
	return a.callContractFunc(a.ctx, code)
}

func (a *MockAPI) GetBlockHeight() int64 {
	//TODO(anarcher):
	return 0
}

func (a *MockAPI) checkFunc(f interface{}) {
	if f == nil {
		panic("this func is nil")
	}

}
*/
