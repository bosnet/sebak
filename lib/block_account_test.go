package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
	"fmt"
	"sync"
)

func TestSaveNewBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	b := testMakeBlockAccount()
	err := b.Save(st)
	if err != nil {
		t.Errorf("failed to save BlockAccount, %v", err)
		return
	}

	exists, err := ExistBlockAccount(st, b.Address)
	if err != nil {
		t.Errorf("failed to get BlockAccount, %v", err)
		return
	}

	if !exists {
		t.Errorf("failed to get BlockAccount, does not exists")
		return
	}
}

func TestSaveExistingBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	b := testMakeBlockAccount()
	b.Save(st)

	if err := b.Deposit(Amount(100), "fake-checkpoint"); err != nil {
		panic(err)
	}
	if err := b.Save(st); err != nil {
		panic(err)
	}

	fetched, _ := GetBlockAccount(st, b.Address)
	if b.Balance != fetched.Balance {
		t.Error("failed to update `BlockAccount.Balance`")
		return
	}
}

func TestSortMultipleBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := testMakeBlockAccount()
		b.Save(st)

		createdOrder = append(createdOrder, b.Address)
	}

	var saved []string
	iterFunc, closeFunc := GetBlockAccountAddressesByCreated(st, false)
	for {
		address, hasNext := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, address)
	}
	closeFunc()

	for i, a := range createdOrder {
		if a != saved[i] {
			t.Error("failed to save `BlockAccount` by creation order")
			break
		}
	}
}

func TestGetSortedBlockAccounts(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := testMakeBlockAccount()
		b.Save(st)

		createdOrder = append(createdOrder, b.Address)
	}

	var saved []string
	iterFunc, closeFunc := GetBlockAccountsByCreated(st, false)
	for {
		ba, hasNext := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, ba.Address)
	}
	closeFunc()

	for i, a := range createdOrder {
		if a != saved[i] {
			t.Error("failed to save `BlockAccount` by creation order")
			break
		}
	}
}

func TestBlockAccountSaveBlockAccountCheckpoints(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	b := testMakeBlockAccount()
	b.Save(st)

	var saved []BlockAccount
	saved = append(saved, *b)
	for i := 0; i < 10; i++ {
		b.Checkpoint = TestGenerateNewCheckpoint()
		b.Save(st)

		saved = append(saved, *b)
	}

	var fetched []BlockAccountCheckpoint
	iterFunc, closeFunc := GetBlockAccountCheckpointByAddress(st, b.Address, false)
	for {
		bac, hasNext := iterFunc()
		if !hasNext {
			break
		}
		fetched = append(fetched, bac)
	}
	closeFunc()

	for i := 0; i < len(saved); i++ {
		if saved[i].Address != fetched[i].Address {
			t.Error("mismatch: Address")
			return
		}
		if saved[i].Balance != fetched[i].Balance {
			t.Error("mismatch: Balance")
			return
		}
		if saved[i].Checkpoint != fetched[i].Checkpoint {
			t.Error("mismatch: Checkpoint")
			return
		}
	}
}
func TestBlockAccountObserver(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	b := testMakeBlockAccount()

	var triggered *BlockAccount
	ObserverFunc := func(args ...interface{}) {
		triggered = args[0].(*BlockAccount)
		wg.Done()
	}
	observer.BlockAccountObserver.On(fmt.Sprintf("saved-%s", b.Address), ObserverFunc)
	defer observer.BlockAccountObserver.Off(fmt.Sprintf("saved-%s", b.Address), ObserverFunc)

	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	b.Save(st)

	wg.Wait()

	if b.Address != triggered.Address {
		t.Error("Address is not match")
		return
	}
	if b.Balance != triggered.Balance {
		t.Error("Balance is not match")
		return
	}
	if b.Checkpoint != triggered.Checkpoint {
		t.Error("Checkpoint is not match")
		return
	}
}
