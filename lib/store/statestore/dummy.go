package statestore

import (
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib"
)

type StorageItem struct {
	Value []byte
}

type StateClone struct {
	db    *sebakstorage.LevelDBBackend
	store *StateStore

	accounts map[string]*sebak.BlockAccount
	objects  map[string]*StateObject
}

func NewStateClone(store *StateStore) *StateClone {

	s := &StateClone{
		db:       store.DBBackend(),
		store:    store,
		accounts: make(map[string]*sebak.BlockAccount),
		objects:  make(map[string]*StateObject),
	}

	return s
}


type StateStore struct {
	db *sebakstorage.LevelDBBackend
}

func NewStateStore(db *sebakstorage.LevelDBBackend) *StateStore {

	s := &StateStore{
		db: db,
	}

	return s
}


type DirtyStatus byte

const (
	StateObjectChanged DirtyStatus = iota
	StateObjectDeleted
)

type StateObject struct {
	Value interface{}
	State DirtyStatus
}