package block

import (
	"fmt"
	"sync"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestSaveNewBlockAccount(t *testing.T) {
	st := storage.NewTestStorage()

	b := TestMakeBlockAccount()
	err := b.Save(st)
	require.NoError(t, err)

	exists, err := ExistsBlockAccount(st, b.Address)
	require.NoError(t, err)
	require.Equal(t, exists, true, "BlockAccount does not exists")
}

func TestSaveExistingBlockAccount(t *testing.T) {
	st := storage.NewTestStorage()

	b := TestMakeBlockAccount()
	b.MustSave(st)

	err := b.Deposit(common.Amount(100))
	require.NoError(t, err)

	err = b.Save(st)
	require.NoError(t, err)

	fetched, _ := GetBlockAccount(st, b.Address)
	require.Equal(t, b.GetBalance(), fetched.GetBalance())
}

func TestSortMultipleBlockAccount(t *testing.T) {
	st := storage.NewTestStorage()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := TestMakeBlockAccount()
		b.MustSave(st)

		createdOrder = append(createdOrder, b.Address)
	}

	var saved []string
	iterFunc, closeFunc := GetBlockAccountAddressesByCreated(st, nil)
	for {
		address, hasNext, _ := iterFunc()
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
	st := storage.NewTestStorage()

	var createdOrder []string
	for i := 0; i < 50; i++ {
		b := TestMakeBlockAccount()
		b.MustSave(st)

		createdOrder = append(createdOrder, b.Address)
	}

	var saved []string
	iterFunc, closeFunc := GetBlockAccountsByCreated(st, nil)
	for {
		ba, hasNext, _ := iterFunc()
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

	st := storage.NewTestStorage()

	b.MustSave(st)

	wg.Wait()

	require.Equal(t, b.Address, triggered.Address)
	require.Equal(t, b.GetBalance(), triggered.GetBalance())
	require.Equal(t, b.SequenceID, triggered.SequenceID)
}
