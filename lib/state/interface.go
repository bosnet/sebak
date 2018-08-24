package state

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/storage"
)

// Reader is the interface that only account and storage state
type Reader interface {
	GetAccount(address string) *block.BlockAccount
	GetStorageItem(address string, key string) *storage.Item
}

// Writer is the interface that account and storage state
type Writer interface {
	SetAccount(account *block.BlockAccount) error
	SetStorageItem(address, key string, item *storage.Item) error
}

// ReadWriter is the interface that groups the Reader and Writer methods.
type ReadWriter interface {
	Reader
	Writer
}

// Committer executes working state to persist

// Hash returns working state's hash root
// Commit performs working state to persisted
// Reset state to underlying State (empty working state)
type Committer interface {
	Hash() ([]byte, error)
	Commit([]byte) error
	Reset() error
}

// Updatable performs to write state and commit this write by hash
//
// It can us used with WorldState.Update(func updater(up Updatable) error) ([]byte,error)
type Updatable interface {
	Writer
	Committer
}
