package stateclone

import (
	"bytes"
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/storage"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

func Test_StateClone_CommitAccount(t *testing.T) {
	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	store := NewStateStore(st)
	clone := NewStateClone(store)

	ba := block.TestMakeBlockAccount()
	ba.Balance = "1000"
	if err := ba.Save(st); err != nil {
		t.Fatal(err)
	}

	am := sebakcommon.Amount(100)
	if err := clone.AccountWithdraw(ba.Address, am, "tx1-tx1"); err != nil {
		t.Fatal(err)
	}

	ba2, err := clone.GetAccount(ba.Address)
	if err != nil {
		t.Fatal(err)
	}

	if ba2.Balance != "900" {
		t.Fatalf("withdraw: have:%v want:%v", ba2.Balance, "900")
	}

	if len(clone.accounts) != 1 {
		t.Fatalf("clone.accounts: have:%v want:%v", len(clone.accounts), 1)
	}

	if hash, err := clone.MakeHashString(); err != nil {
		t.Fatal(err)
	} else {
		t.Log("hash:", hash)
		if hash == "" {
			t.Fatal("clone.hash is nil")
		}
	}

	if err := clone.Commit(); err != nil {
		t.Fatal(err)
	}

	{
		ba3, err := block.GetBlockAccount(st, ba.Address)
		if err != nil {
			t.Fatal(err)
		}

		if ba3.Balance != ba2.Balance {
			t.Fatalf("balance: have:%v want:%v", ba3.Balance, ba2.Balance)
		}
	}
}

func Test_StateStore_CommitStorageItem(t *testing.T) {
	var (
		addr = "hello"
		key  = "key1"
	)

	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	store := NewStateStore(st)
	clone := NewStateClone(store)

	item := &storage.StorageItem{
		Value: []byte("item1"),
	}

	if err := clone.PutStorageItem(addr, key, item); err != nil {
		t.Fatal(err)
	}

	if len(clone.objects) != 1 {
		t.Fatalf("clone.objects have:%v want:%v", len(clone.objects), 1)
	}

	{
		item1, err := clone.GetStorageItem(addr, key)
		if err != nil {
			t.Fatal(err)
		}
		if item1 != item {
			t.Fatalf("item1 have:%v want:%v", item1, item)
		}
	}

	{
		h1, err := clone.MakeHashString()
		if err != nil {
			t.Fatal(err)
		}
		if err := clone.PutStorageItem(addr, key, item); err != nil {
			t.Fatal(err)
		}

		h2, err := clone.MakeHashString()
		if err != nil {
			t.Fatal(err)
		}
		if h1 != h2 {
			t.Fatalf("hash: have:%v want:%v", h2, h1)
		}

	}

	if err := clone.Commit(); err != nil {
		t.Fatal(err)
	}

	{
		item1, err := store.GetStorageItem(addr, key)
		if err != nil {
			t.Fatal(err)
		}
		if item1 == nil {
			t.Fatalf("item1 from store have:%v want:%v", item1, item)
		}

		if !bytes.Equal(item1.Value, item.Value) {
			t.Fatalf("item1.Value: have:%v want:%v", item1.Value, item.Value)
		}
	}
}
