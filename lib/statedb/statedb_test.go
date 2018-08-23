package statedb

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"fmt"
	"github.com/google/uuid"
	"github.com/magiconair/properties/assert"
	"sync"
	"testing"
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
		assert.Equal(t, gotValueHash, valueHash)
	}

	{
		t := trie.NewTrie(Root, trie.NewEthDatabase(st))
		key := []byte("key")
		value := []byte("value")
		t.TryUpdate(key, value)
		v, _ := t.TryGet(key)
		fmt.Println(string(v))
	}
}
func TestSaveNewBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	statedb := TestNewStateDB(st)
	addr := TestCreateAccount(statedb)

	root, err := statedb.CommitTrie()
	if err != nil {
		t.Errorf("failed to Commit Trie, %v", err)
		return
	}

	err = statedb.CommitDB(root)
	if err != nil {
		t.Errorf("failed to Commit DB, %v", err)
		return
	}

	if statedb.ExistAccount(addr) == false {
		t.Errorf("failed to get BlockAccount, %v", err)
		return
	}

}
func TestSaveExistingBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	statedb := TestNewStateDB(st)
	addr := TestCreateAccount(statedb)

	statedb.AddBalanceWithCheckpoint(addr, sebakcommon.Amount(100), "fake-checkpoint")
	storeBalance := statedb.GetBalance(addr)

	root, err := statedb.CommitTrie()
	if err != nil {
		t.Errorf("failed to Commit Trie, %v", err)
		return
	}

	err = statedb.CommitDB(root)
	if err != nil {
		t.Errorf("failed to Commit DB, %v", err)
		return
	}

	statedb = TestNewStateDBWithRoot(st, root)

	gotBalance := statedb.GetBalance(addr)

	if gotBalance != storeBalance {
		t.Error("failed to update `BlockAccount.Balance`")
		return
	}

	ba, err := block.GetBlockAccount(st, addr)
	if err != nil {
		t.Error(err)
		return
	}
	if ba.Balance != storeBalance {
		t.Error("failed to update `BlockAccount.Balance`")
		return
	}
}

func TestSortMultipleBlockAccount(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	statedb := TestNewStateDB(st)

	var createdOrder []string
	for i := 0; i < 50; i++ {
		addr := TestCreateAccount(statedb)
		createdOrder = append(createdOrder, addr)
		root, err := statedb.CommitTrie()
		if err != nil {
			t.Errorf("failed to Commit Trie, %v", err)
			return
		}

		err = statedb.CommitDB(root)
		if err != nil {
			t.Errorf("failed to Commit DB, %v", err)
			return
		}
	}

	var saved []string
	iterFunc, closeFunc := block.GetBlockAccountAddressesByCreated(st, false)
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
	statedb := TestNewStateDB(st)

	var createdOrder []string
	for i := 0; i < 50; i++ {
		addr := TestCreateAccount(statedb)
		createdOrder = append(createdOrder, addr)
		root, err := statedb.CommitTrie()
		if err != nil {
			t.Errorf("failed to Commit Trie, %v", err)
			return
		}

		err = statedb.CommitDB(root)
		if err != nil {
			t.Errorf("failed to Commit DB, %v", err)
			return
		}
	}

	var saved []string
	iterFunc, closeFunc := block.GetBlockAccountsByCreated(st, false)
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
	statedb := TestNewStateDB(st)

	addr := TestCreateAccount(statedb)
	root, err := statedb.CommitTrie()
	if err != nil {
		t.Errorf("failed to Commit Trie, %v", err)
		return
	}

	err = statedb.CommitDB(root)
	if err != nil {
		t.Errorf("failed to Commit DB, %v", err)
		return
	}

	var saved []block.BlockAccount
	b, err := block.GetBlockAccount(st, addr)
	if err != nil {
		t.Error(err)
	}
	saved = append(saved, *b)
	for i := 0; i < 10; i++ {
		statedb := TestNewStateDBWithRoot(st, root)
		checkpoint := uuid.New().String()
		statedb.SetCheckpoint(addr, checkpoint)
		root, err := statedb.CommitTrie()
		if err != nil {
			t.Errorf("failed to Commit Trie, %v", err)
			return
		}

		err = statedb.CommitDB(root)
		if err != nil {
			t.Errorf("failed to Commit DB, %v", err)
			return
		}
		b, err := block.GetBlockAccount(st, addr)
		if err != nil {
			t.Error(err)
		}
		saved = append(saved, *b)
	}

	var fetched []block.BlockAccountCheckpoint
	iterFunc, closeFunc := block.GetBlockAccountCheckpointByAddress(st, addr, false)
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
			t.Error("misatch: Checkpoint")
			return
		}
	}
}

func TestBlockAccountObserver(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	statedb := TestNewStateDB(st)

	addr := TestCreateAccount(statedb)

	var triggered *block.BlockAccount
	ObserverFunc := func(args ...interface{}) {
		triggered = args[0].(*block.BlockAccount)
		wg.Done()
	}
	observer.BlockAccountObserver.On(fmt.Sprintf("address-%s", addr), ObserverFunc)
	defer observer.BlockAccountObserver.Off(fmt.Sprintf("address-%s", addr), ObserverFunc)

	root, err := statedb.CommitTrie()
	if err != nil {
		t.Errorf("failed to Commit Trie, %v", err)
		return
	}

	err = statedb.CommitDB(root)
	if err != nil {
		t.Errorf("failed to Commit DB, %v", err)
		return
	}

	wg.Wait()

	if addr != triggered.Address {
		t.Error("Address is not match")
		return
	}
	if statedb.GetBalance(addr) != triggered.Balance {
		t.Error("Balance is not match")
		return
	}
	if statedb.GetCheckPoint(addr) != triggered.Checkpoint {
		t.Error("Checkpoint is not match")
		return
	}
}
