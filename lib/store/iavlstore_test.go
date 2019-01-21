package store

import (
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	//"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
	"testing"
)

func TestIAVL(t *testing.T) {
	db := dbm.NewMemDB()
	st, err := NewIAVLStore(db, PruneNothing)
	require.NoError(t, err)

	st.Set([]byte("hello"), []byte("bye"))
	g := st.Get([]byte("hello"))

	t.Log(string(g))

	st.Commit()

	r := st.Query(RequestQuery{
		Path:  "/key",
		Data:  []byte("hello"),
		Prove: true,
	})

	t.Log(string(r.Value))
	t.Log(r.Proof)

}

func TestT(t *testing.T) {
	db := dbm.NewMemDB()
	st := store.NewCommitMultiStore(db)

	key1 := sdk.NewKVStoreKey("store1")
	key2 := sdk.NewKVStoreKey("store2")

	st.MountStoreWithDB(key1, sdk.StoreTypeIAVL, nil)
	st.MountStoreWithDB(key2, sdk.StoreTypeIAVL, nil)

	err := st.LoadLatestVersion()
	require.Nil(t, err)

	st1 := st.GetCommitKVStore(key1)
	st1.Set([]byte("hello"), []byte("buy"))

	st1.Commit()

	st.Commit()

	g := st1.Get([]byte("hello"))
	t.Log(string(g))
	cid := st.Commit()
	t.Log(cid)
	cid = st.Commit()

	t.Log(cid)

}

//func TestI (t *testing.T){
//	db := dbm.NewMemDB()
//
//	sdk.NewTransientStoreKey()
//
//	st, err := store.LoadIAVLStore(db, sdk.CommitID{}, sdk.PruneNothing)
//	require.NoError(t, err)
//
//	st1 := st.(store.CommitKVStore)
//	st1.Set([]byte("hello"), []byte("buy"))
//	g := st1.Get([]byte("hello"))
//	t.Log(string(g))
//	cid := st.Commit()
//	t.Log(cid)
//	cid = st.Commit()
//}

func TestVerifyMultiStoreQueryProof(t *testing.T) {
	// Create main tree for testing.
	db := dbm.NewMemDB()
	st := store.NewCommitMultiStore(db)
	iavlStoreKey := sdk.NewKVStoreKey("iavlStoreKey")

	st.MountStoreWithDB(iavlStoreKey, sdk.StoreTypeIAVL, nil)
	st.LoadVersion(0)

	iavlStore := st.GetCommitKVStore(iavlStoreKey)
	iavlStore.Set([]byte("MYKEY"), []byte("MYVALUE"))
	iavlStore.Commit()
	cid := st.Commit()

	// Get Proof
	res := st.Query(abci.RequestQuery{
		Path:  "/iavlStoreKey/key", // required path to get key/value+proof
		Data:  []byte("MYKEY"),
		Prove: true,
	})
	require.NotNil(t, res.Proof)

	// Verify proof.
	prt := store.DefaultProofRuntime()
	err := prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY", []byte("MYVALUE"))
	require.Nil(t, err)

	// Verify proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY", []byte("MYVALUE"))
	require.Nil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY_NOT", []byte("MYVALUE"))
	require.NotNil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY/MYKEY", []byte("MYVALUE"))
	require.NotNil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "iavlStoreKey/MYKEY", []byte("MYVALUE"))
	require.NotNil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/MYKEY", []byte("MYVALUE"))
	require.NotNil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY", []byte("MYVALUE_NOT"))
	require.NotNil(t, err)

	// Verify (bad) proof.
	err = prt.VerifyValue(res.Proof, cid.Hash, "/iavlStoreKey/MYKEY", []byte(nil))
	require.NotNil(t, err)
}
