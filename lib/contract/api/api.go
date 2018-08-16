package api

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/contract/value"
)

type API interface {
	Helloworld(string) (string, error)
	GetStorageItem(string) (*storage.StorageItem, error)
	PutStorageItem(string, *storage.StorageItem) error
	GetBalance() (sebakcommon.Amount, error)
	CallContract(*payload.ExecCode) (*value.Value, error)
	GetBlockHeight() int64
}
