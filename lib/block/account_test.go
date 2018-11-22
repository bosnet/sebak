package block

import (
	"math/rand"
	"testing"

	"boscoin.io/sebak/lib/common"
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

func TestBlockAccountSaveBlockAccountSequenceIDs(t *testing.T) {
	st := storage.NewTestStorage()

	b := TestMakeBlockAccount()
	b.MustSave(st)

	expectedSavedLength := 10
	var saved []BlockAccount
	saved = append(saved, *b)
	for i := 0; i < expectedSavedLength-len(saved); i++ {
		b.SequenceID = rand.Uint64()
		b.MustSave(st)

		saved = append(saved, *b)
	}

	var fetched []BlockAccountSequenceID
	options := storage.NewDefaultListOptions(false, nil, uint64(expectedSavedLength))
	iterFunc, closeFunc := GetBlockAccountSequenceIDByAddress(st, b.Address, options)
	for {
		bac, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}
		fetched = append(fetched, bac)
	}
	closeFunc()

	require.Equal(t, len(saved), len(fetched))
	for i, b := range saved {
		require.Equal(t, b.Address, fetched[i].Address)
		require.Equal(t, b.GetBalance(), fetched[i].Balance)
		require.Equal(t, b.SequenceID, fetched[i].SequenceID)
	}
}
