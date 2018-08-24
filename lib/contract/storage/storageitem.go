package storage

import (
	"fmt"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

const (
	StorageItemKeyPrefix = "tc-si-" // tc-si-{address}-{key}
)

// Storage is a storage for contract

type StorageItem struct {
	Address string
	Key     string
	Value   []byte
}

func NewStorageItem(addr, key string) *StorageItem {
	item := &StorageItem{
		Address: addr,
		Key:     key,
	}

	return item
}

func (s *StorageItem) Save(st sebakstorage.DBBackend) error {
	dbkey := GetStorageItemDBKey(s.Address, s.Key)
	has, err := st.Has(dbkey)
	if err != nil {
		return err
	}

	{
		var err error
		if has {
			err = st.Set(dbkey, s)
		} else {
			err = st.New(dbkey, s)
		}
		return err
	}
}

func GetStorageItemDBKey(addr, key string) string {
	return fmt.Sprintf("%s%s-%s", StorageItemKeyPrefix, addr, key)
}

func GetStorageItem(st sebakstorage.DBBackend, addr, key string) (*StorageItem, error) {
	var item *StorageItem
	if err := st.Get(GetStorageItemDBKey(addr, key), &item); err != nil {
		if err == sebakerror.ErrorStorageRecordDoesNotExist {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}
