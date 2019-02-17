package eth

import (
	"boscoin.io/sebak/lib/state/tree"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

type Trie struct {
	trie *trie.Trie
}

var _ tree.Tree = (*Trie)(nil) // Implements tree/Tree interface

func NewTrie(root []byte, db *EthDB) (*Trie, error) {
	hash := ethcommon.BytesToHash(root)
	trieDB := trie.NewDatabase(db)
	ethTrie, err := trie.New(hash, trieDB)
	if err != nil {
		return nil, err
	}

	t := &Trie{
		trie: ethTrie,
	}
	return t, nil
}

func (t *Trie) Hash() []byte {
	hash := t.trie.Hash()
	return hash.Bytes()
}

func (t *Trie) Commit() ([]byte, error) {
	hash, err := t.trie.Commit(nil)
	if err != nil {
		return nil, err
	}
	return hash.Bytes(), nil
}

func (t *Trie) Set(key, value []byte) error {
	if err := t.trie.TryUpdate(key, value); err != nil {
		return err
	}
	return nil
}

func (t *Trie) Get(key []byte) ([]byte, error) {
	value, err := t.trie.TryGet(key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (t *Trie) Delete(key []byte) error {
	if err := t.trie.TryDelete(key); err != nil {
		return err
	}
	return nil
}
