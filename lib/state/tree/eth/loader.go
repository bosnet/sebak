package eth

import (
	"boscoin.io/sebak/lib/state/tree"
	"boscoin.io/sebak/lib/storage"
)

type Loader struct {
	db *sebakstorage.LevelDBBackend
}

var _ tree.Loader = (*Loader)(nil)

func NewLoader(db *sebakstorage.LevelDBBackend) *Loader {
	l := &Loader{
		db: db,
	}
	return l
}

func (l *Loader) loadTree(hash []byte) (*Trie, error) {

	ethdb := NewEthDB(l.db)
	trie, err := NewTrie(hash, ethdb)
	if err != nil {
		return nil, err
	}
	return trie, nil
}

func (l *Loader) LoadMutableTree(hash []byte) (tree.MutableTree, error) {
	return l.loadTree(hash)
}

func (l *Loader) LoadImmutableTree(hash []byte) (tree.ImmutableTree, error) {
	return l.loadTree(hash)
}
