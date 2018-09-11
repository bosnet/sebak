package block

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestSaveNewBlockAccount(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	b := TestMakeBlockAccount()
	err := b.Save(st)
	require.Nil(t, err)

	exists, err := ExistBlockAccount(st, b.Address)
	require.Nil(t, err)
	require.Equal(t, exists, true, "BlockAccount does not exists")
}

func TestSaveExistingBlockAccount(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	b := TestMakeBlockAccount()
	b.Save(st)

	err := b.Deposit(common.Amount(100))
	require.Nil(t, err)

	err = b.Save(st)
	require.Nil(t, err)

	fetched, _ := GetBlockAccount(st, b.Address)
	require.Equal(t, b.GetBalance(), fetched.GetBalance())
}

func TestSortMultipleBlockAccount(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := TestMakeBlockAccount()
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
		require.Equal(t, a, saved[i], "Blockaccount are not saved in the order they are created")
	}
}

func TestGetSortedBlockAccounts(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := TestMakeBlockAccount()
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
		require.Equal(t, a, saved[i], "Blockaccount are not saved in the order they are created")
	}
}

func TestBlockAccountSaveBlockAccountSequenceIDs(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	b := TestMakeBlockAccount()
	b.Save(st)

	var saved []BlockAccount
	saved = append(saved, *b)
	for i := 0; i < 10; i++ {
		b.SequenceID = rand.Uint64()
		b.Save(st)

		saved = append(saved, *b)
	}

	var fetched []BlockAccountSequenceID
	iterFunc, closeFunc := GetBlockAccountSequenceIDByAddress(st, b.Address, false)
	for {
		bac, hasNext := iterFunc()
		if !hasNext {
			break
		}
		fetched = append(fetched, bac)
	}
	closeFunc()

	for i := 0; i < len(saved); i++ {
		require.Equal(t, saved[i].Address, fetched[i].Address)
		require.Equal(t, saved[i].GetBalance(), fetched[i].Balance)
		require.Equal(t, saved[i].SequenceID, fetched[i].SequenceID)
	}
}
func TestBlockAccountObserver(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	b := TestMakeBlockAccount()

	var triggered *BlockAccount
	ObserverFunc := func(args ...interface{}) {
		triggered = args[0].(*BlockAccount)
		wg.Done()
	}
	observer.BlockAccountObserver.On(fmt.Sprintf("address-%s", b.Address), ObserverFunc)
	defer observer.BlockAccountObserver.Off(fmt.Sprintf("address-%s", b.Address), ObserverFunc)

	st, _ := storage.NewTestMemoryLevelDBBackend()

	b.Save(st)

	wg.Wait()

	require.Equal(t, b.Address, triggered.Address)
	require.Equal(t, b.GetBalance(), triggered.GetBalance())
	require.Equal(t, b.SequenceID, triggered.SequenceID)
}
