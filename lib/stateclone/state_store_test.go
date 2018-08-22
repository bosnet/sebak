package stateclone

import (
	"bytes"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/storage"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

func Test_StateStore_GetAccount(t *testing.T) {
	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	store := NewStateStore(st)

	ba := block.TestMakeBlockAccount()
	if err := ba.Save(st); err != nil {
		t.Fatal(err)
	}

	ba2, err := store.GetAccount(ba.Address)
	if err != nil {
		t.Fatal(err)
	}

	if ba2 == nil {
		t.Fatal("account from state store is nil!")
	}

	if ba2.Address != ba.Address {
		t.Fatalf("address have:%v want:%v", ba2.Address, ba.Address)

	}

	if ba2.Balance != ba.Balance {
		t.Fatalf("balance have:%v want:%v", ba2.Balance, ba.Balance)

	}
}

func Test_StateStore_StorageGet(t *testing.T) {
	addr := "helloworld"
	keyname := "key1"
	itemkey := getContractStorageItemKey(addr, keyname)
	item1 := &storage.StorageItem{
		Value: []byte("item1"),
	}

	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	if err := st.New(itemkey, item1); err != nil {
		t.Fatal(err)
	}

	store := NewStateStore(st)

	item2, err := store.GetStorageItem(addr, keyname)
	if err != nil {
		t.Fatal(err)
	}

	if item2 == nil {
		t.Fatal("item is nil")
	}

	if !bytes.Equal(item1.Value, item2.Value) {
		t.Fatalf("item have:%v want:%v", item2.Value, item1.Value)
	}

}
