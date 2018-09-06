package trie

import (
	gocommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"

	"boscoin.io/sebak/lib/common"
)

type Trie struct {
	trie.Trie
	DB *trie.Database
}

func NewTrie(root common.Hash, db *EthDatabase) *Trie {
	triedb := trie.NewDatabase(db)
	tr, err := trie.New(gocommon.Hash(root), triedb)
	if err != nil {
		panic(err)
	}
	return &Trie{
		Trie: *tr,
		DB:   triedb,
	}
}

func (t *Trie) CommitDB(root common.Hash) (err error) {
	return t.DB.Commit(root, false)
}
