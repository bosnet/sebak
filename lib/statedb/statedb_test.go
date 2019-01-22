package statedb

import (
	"boscoin.io/sebak/lib/statedb/store"
	"testing"

	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestStateDB(t *testing.T) {
	var commitID = store.CommitID{}
	db := dbm.NewMemDB()

	var keyHash = []byte("key")
	var valueHash = []byte("value")

	var err error

	{
		stateDB := New(db, []byte{}, 0)
		stateDB.GetOrNewStateObject("dummy")
		stateDB.SetState("dummy", keyHash, valueHash)
		commitID = stateDB.Commit()
		if err != nil {
			t.Error(err)
		}
		gotValueHash := stateDB.GetState("dummy", keyHash)
		require.Equal(t, gotValueHash, valueHash)
	}

	{
		stateDB := New(db, commitID.Hash, commitID.Version)
		gotValueHash := stateDB.GetState("dummy", keyHash)
		require.Equal(t, gotValueHash, valueHash)
	}
}
