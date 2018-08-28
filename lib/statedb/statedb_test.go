package statedb

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestStateDB(t *testing.T) {
	var Root = sebakcommon.Hash{}
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	var keyHash = sebakcommon.BytesToHash(sebakcommon.MakeHash([]byte("key")))
	var value = []byte("value")

	var err error

	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		stateDB.GetOrNewStateObject("dummy")
		stateDB.SetState("dummy", keyHash, value)
		Root, err = stateDB.CommitTrie()
		if err != nil {
			t.Error(err)
		}
		stateDB.CommitDB(Root)
	}

	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		gotValueHash := stateDB.GetState("dummy", keyHash)
		assert.Equal(t, gotValueHash, value)
	}
	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		stateDB.SetCode("dummy", []byte("codecode"))
		Root, err = stateDB.CommitTrie()
		if err != nil {
			t.Error(err)
		}
		stateDB.CommitDB(Root)
	}
	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		encoded := stateDB.GetCode("dummy")
		t.Log(encoded)
	}

	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		stateDB.SetCode("dummy", []byte("codecode2222"))
		Root, err = stateDB.CommitTrie()
		if err != nil {
			t.Error(err)
		}
		stateDB.CommitDB(Root)
	}
	{
		edb := trie.NewEthDatabase(st)
		var a []byte
		a, err = edb.Get(sebakcommon.MakeHash([]byte("codecode2222")))
		t.Log(a)
	}
}
