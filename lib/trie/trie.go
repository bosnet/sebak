package trie

import (
	"boscoin.io/sebak/lib/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

type Trie struct {
	trie.Trie
	DB *trie.Database
}

func NewTrie(root sebakcommon.Hash, db *EthDatabase) *Trie {
	triedb := trie.NewDatabase(db)
	tr, err := trie.New(common.Hash(root), triedb)
	if err != nil {
		panic(err)
	}
	return &Trie{
		Trie: *tr,
		DB:   triedb,
	}
}

func (t *Trie) CommitDB(root sebakcommon.Hash) {
	t.DB.Commit(root, false)
}
