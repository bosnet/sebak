package api

import "boscoin.io/sebak/lib/contract/storage"

type API interface {
	Helloworld(string) (string, error)
	GetStorageItem(string) (*storage.StorageItem, error)
	PutStorageItem(string, *storage.StorageItem) error
}
