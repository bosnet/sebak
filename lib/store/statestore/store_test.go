package statestore

import (
	"bytes"
	"testing"

	"boscoin.io/sebak/lib"
)

func Test_StateStore_GetAccount(t *testing.T) {
	var (
		address                 = "helloworld"
		balance    sebak.Amount = 1000
		checkpoint              = "tx1-tx1"
	)

	ba := sebak.NewBlockAccount(address, balance, checkpoint)
	if err := ba.Save(testLevelDBBackend); err != nil {
		t.Error(err)
		return
	}

	account, err := testStateStore.GetAccount(address)
	if err != nil {
		t.Error(err)
		return
	}
	if account == nil {
		t.Errorf("account is nil")
		return
	}

	if account.Address != ba.Address {
		t.Errorf("address have:%v want:%v", account.Address, ba.Address)
		return
	}
	if account.Balance != ba.Balance {
		t.Errorf("balance have:%v want:%v", account.Balance, ba.Balance)
		return
	}

}

func Test_StateStore_StorageGet(t *testing.T) {
	itemKey := getContractStorageItemKey("helloworld", "key2")
	item := &StorageItem{
		Value: []byte("item1"),
	}

	if err := testLevelDBBackend.New(itemKey, item); err != nil {
		t.Error(err)
		return
	}

	itemR, err := testStateStore.GetStorageItem("helloworld", "key2")
	if err != nil {
		t.Error(err)
		return
	}

	if itemR == nil {
		t.Errorf("item result is nil")
		return
	}
	if !bytes.Equal(itemR.Value, item.Value) {
		t.Errorf("item have:%v want:%v", itemR.Value, item.Value)
		return
	}
}
