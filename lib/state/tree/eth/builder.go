package eth

import (
	"boscoin.io/sebak/lib/state/tree"
	"boscoin.io/sebak/lib/storage"
)

type Builder struct {
	db *sebakstorage.LevelDBBackend
}

var _ tree.Builder = (*Builder)(nil)

func NewBuilder(db *sebakstorage.LevelDBBackend) *Builder {
	b := &Builder{
		db: db,
	}
	return b
}

func (b *Builder) Build(hash []byte) (tree.Tree, error) {
	ethdb := NewEthDB(b.db)
	trie, err := NewTrie(hash, ethdb)
	if err != nil {
		return nil, err
	}
	return trie, nil
}
