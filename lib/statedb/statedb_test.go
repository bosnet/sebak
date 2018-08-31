package statedb

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"

	"github.com/stretchr/testify/require"
)

func TestStateDB(t *testing.T) {
	var Root = sebakcommon.Hash{}
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	var keyHash = sebakcommon.BytesToHash(sebakcommon.MakeHash([]byte("key")))
	var valueHash = sebakcommon.BytesToHash(sebakcommon.MakeHash([]byte("value")))

	var err error

	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		stateDB.GetOrNewStateObject("dummy")
		stateDB.SetState("dummy", keyHash, valueHash)
		Root, err = stateDB.CommitTrie()
		if err != nil {
			t.Error(err)
		}
		stateDB.CommitDB(Root)
	}

	{
		stateDB := New(Root, trie.NewEthDatabase(st))
		gotValueHash := stateDB.GetState("dummy", keyHash)
		require.Equal(t, gotValueHash, valueHash)
	}
}
