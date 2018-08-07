package statestore

import (
	"bytes"
	"testing"

	sebak "boscoin.io/sebak/lib"
)

func Test_StateClone_CommitAccount(t *testing.T) {
	var (
		addr                    = "account1"
		balance    sebak.Amount = 1000
		checkpoint              = "tx1-tx1"
	)
	ba := sebak.NewBlockAccount(addr, balance, checkpoint)
	if err := ba.Save(testLevelDBBackend); err != nil {
		t.Fatal(err)
	}

	clone := NewStateClone(testStateStore)

	wdAmount := sebak.Amount(100)
	if err := clone.AccountWithdraw(addr, wdAmount, checkpoint); err != nil {
		t.Fatal(err)
	}

	addr2, err := clone.GetAccount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if sebak.MustAmountFromString(addr2.Balance) != balance.MustSub(wdAmount) {
		t.Fatalf("withdraw: have:%v want:%v", addr2.Balance, balance.MustSub(wdAmount))
	}

	if len(clone.accounts) != 1 {
		t.Fatalf("clone.accounts have: %v want: %v", len(clone.accounts), 1)
	}

	if hash, err := clone.MakeHashString(); err != nil {
		t.Fatal(err)
	} else {
		t.Log("hash:", hash)
		if hash == "" {
			t.Fatalf("clone.hash is nil")
		}
	}

	if err := clone.Commit(); err != nil {
		t.Fatal(err)
	}

	{
		ba, err := sebak.GetBlockAccount(testLevelDBBackend, addr)
		if err != nil {
			t.Fatal(err)
		}
		if ba.Balance != addr2.Balance {
			t.Fatalf("balance: have: %v want: %v", ba.Balance, addr2.Balance)
		}
	}
}

func Test_StateClone_CommitStorageItem(t *testing.T) {
	clone := NewStateClone(testStateStore)

	item := &StorageItem{
		Value: []byte("item1"),
	}

	if err := clone.PutStorageItem("hello", "key1", item); err != nil {
		t.Fatal(err)
	}

	if len(clone.objects) != 1 {
		t.Fatalf("clone.objects have:%v want:%v", len(clone.objects), 1)
	}

	if item1, err := clone.GetStorageItem("hello", "key1"); err != nil {
		t.Fatal(err)
	} else if err == nil {
		if item1 != item {
			t.Fatalf("item1 have:%v want:%v", item1, nil)
		}
	}

	{
		h1, err := clone.MakeHashString()
		if err != nil {
			t.Fatal(err)
		}
		if err := clone.PutStorageItem("hello", "key1", item); err != nil {
			t.Fatal(err)
		}

		h2, err := clone.MakeHashString()
		if err != nil {
			t.Fatal(err)
		}

		if h1 != h2 {
			t.Fatalf("hash1:%v hash2:%v", h1, h2)
		}
	}

	if err := clone.Commit(); err != nil {
		t.Fatal(err)
	}

	if item1, err := testStateStore.GetStorageItem("hello", "key1"); err != nil {
		t.Fatal(err)
	} else if err == nil {
		if item1 == nil {
			t.Fatalf("item1 have:%v want:%v", item1, item)
		}
		if !bytes.Equal(item1.Value, item.Value) {
			t.Fatalf("item1.Value have:%v want:%v", item1.Value, item.Value)
		}
	}
}
