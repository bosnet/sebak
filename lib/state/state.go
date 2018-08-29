package state

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/block"
	cstorage "boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/state/tree"
)

type State struct {
	tree       tree.ImmutableTree
	treeLoader tree.Loader
	hash       []byte // Current state's hash

	logger logging.Logger
}

var _ Reader = (*State)(nil)

func NewState(treeLoader tree.Loader, hash []byte) *State {
	tree, err := treeLoader.LoadMutableTree(hash)
	if err != nil {
		panic(err)
	}

	s := &State{
		treeLoader: treeLoader,
		tree:       tree,
		logger:     logging.New("module", "state"),
	}

	return s
}

func (s *State) GetAccount(address string) (*block.BlockAccount, error) {
	value, err := s.tree.Get([]byte(address))
	if err != nil {
		return nil, err
	}

	account := &block.BlockAccount{}
	if err := account.Deserialize(value); err != nil {
		return nil, err
	}

	return account, nil
}

func (s *State) GetStorageItem(address, key string) (*cstorage.Item, error) {
	value, err := s.tree.Get([]byte(address))
	if err != nil {
		return nil, err
	}

	item := &cstorage.Item{}
	if err := item.Deserialize(value); err != nil {
		return nil, err
	}

	return item, nil
}
