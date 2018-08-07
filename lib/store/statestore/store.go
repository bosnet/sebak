package statestore

import (
	"boscoin.io/sebak/lib"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

type StateStore struct {
	db *sebakstorage.LevelDBBackend
}

func NewStateStore(db *sebakstorage.LevelDBBackend) *StateStore {

	s := &StateStore{
		db: db,
	}

	return s
}

func (s *StateStore) DBBackend() *sebakstorage.LevelDBBackend {
	return s.db
}

func (s *StateStore) GetAccount(addr string) (*sebak.BlockAccount, error) {
	ba, err := sebak.GetBlockAccount(s.db, addr)
	if err != nil {
		return nil, err
	}

	return ba, nil
}

func (s *StateStore) GetStorageItem(addr, key string) (*StorageItem, error) {
	itemKey := getContractStorageItemKey(addr, key)

	var item *StorageItem
	if err := s.db.Get(itemKey, &item); err != nil {
		if err == sebakerror.ErrorStorageRecordDoesNotExist {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (s *StateStore) GetDeployCode(addr string) (*payload.DeployCode, error) {
	k := getContractCodeKey(addr)

	var code *payload.DeployCode
	if err := s.db.Get(k, code); err != nil {
		return nil, err
	}
	return code, nil
}
