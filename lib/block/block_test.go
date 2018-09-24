package block

import (
	"math/rand"
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestBlockConfirmedOrdering(t *testing.T) {
	st := storage.NewTestStorage()

	var inserted []Block
	for i := 0; i < 10; i++ {
		bk := TestMakeNewBlock([]string{})
		bk.Height = uint64(i)
		require.Nil(t, bk.Save(st))
		inserted = append(inserted, bk)
	}

	{ // reverse = false
		var fetched []Block
		iterFunc, closeFunc := GetBlocksByConfirmed(st, storage.NewDefaultListOptions(false, nil, 10))
		for {
			t, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			fetched = append(fetched, t)
		}
		closeFunc()

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[i].Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // reverse = true
		var fetched []Block
		iterFunc, closeFunc := GetBlocksByConfirmed(st, storage.NewDefaultListOptions(true, nil, 10))
		for {
			t, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			fetched = append(fetched, t)
		}
		closeFunc()

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[len(fetched)-1-i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[len(fetched)-1-i].Serialize()
			require.Equal(t, s, rs)
		}
	}
}

func TestBlockHeightOrdering(t *testing.T) {
	st := storage.NewTestStorage()

	// save Block, but Height will be shuffled
	numberOfBlocks := 10
	inserted := make([]Block, numberOfBlocks)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for _, i := range r.Perm(numberOfBlocks) {
		bk := TestMakeNewBlock([]string{})
		bk.Height = uint64(i)
		require.Nil(t, bk.Save(st))
		inserted[i] = bk
	}

	{
		var fetched []Block
		for i := 0; i < numberOfBlocks; i++ {
			b, err := GetBlockByHeight(st, uint64(i))
			require.Nil(t, err)
			fetched = append(fetched, b)
		}

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[i].Serialize()
			require.Equal(t, s, rs)
		}
	}
}

// TestMakeGenesisBlock basically tests MakeGenesisBlock can make genesis block,
// and further with genesis block, genesis account can be found.
func TestMakeGenesisBlock(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	kp, _ := keypair.Random()
	balance := common.Amount(100)
	account := NewBlockAccount(kp.Address(), balance)
	err := account.Save(st)
	require.Nil(t, err)

	bk, err := MakeGenesisBlock(st, *account, networkID)
	require.Nil(t, err)
	require.Equal(t, uint64(1), bk.Height)
	require.Equal(t, 1, len(bk.Transactions))
	require.Equal(t, uint64(0), bk.Round.Number)
	require.Equal(t, "", bk.Round.BlockHash)
	require.Equal(t, uint64(0), bk.Round.BlockHeight)
	require.Equal(t, "", bk.Proposer)
	require.Equal(t, common.GenesisBlockConfirmedTime, bk.Confirmed)

	// transaction
	{
		exists, err := ExistsBlockTransaction(st, bk.Transactions[0])
		require.Nil(t, err)
		require.True(t, exists)
	}

	bt, err := GetBlockTransaction(st, bk.Transactions[0])
	require.Nil(t, err)

	require.Equal(t, account.SequenceID, bt.SequenceID)
	require.Equal(t, common.Amount(0), bt.Fee)
	require.Equal(t, 1, len(bt.Operations))
	require.Equal(t, account.Address, bt.Source)
	require.Equal(t, bk.Hash, bt.Block)

	// operation
	{
		exists, err := ExistsBlockOperation(st, bt.Operations[0])
		require.Nil(t, err)
		require.True(t, exists)
	}
	bo, err := GetBlockOperation(st, bt.Operations[0])
	require.Nil(t, err)
	require.Equal(t, bt.Hash, bo.TxHash)
	require.Equal(t, transaction.OperationCreateAccount, bo.Type)
	require.Equal(t, account.Address, bo.Source)
	require.Equal(t, account.Address, bo.Target)
	require.Equal(t, account.Balance, bo.Amount)
}

func TestMakeGenesisBlockOverride(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	{ // create genesis block
		kp, _ := keypair.Random()
		balance := common.Amount(100)
		account := NewBlockAccount(kp.Address(), balance)
		err := account.Save(st)
		require.Nil(t, err)

		bk, err := MakeGenesisBlock(st, *account, networkID)
		require.Nil(t, err)
		require.Equal(t, uint64(1), bk.Height)
	}

	{ // try again to create genesis block
		kp, _ := keypair.Random()
		balance := common.Amount(100)
		account := NewBlockAccount(kp.Address(), balance)
		err := account.Save(st)
		require.Nil(t, err)

		_, err = MakeGenesisBlock(st, *account, networkID)
		require.Equal(t, errors.ErrorBlockAlreadyExists, err)
	}
}

func TestMakeGenesisBlockFindGenesisAccount(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	// create genesis block
	kp, _ := keypair.Random()
	balance := common.Amount(100)
	account := NewBlockAccount(kp.Address(), balance)
	account.Save(st)

	{
		bk, err := MakeGenesisBlock(st, *account, networkID)
		require.Nil(t, err)
		require.Equal(t, uint64(1), bk.Height)
	}

	// find genesis account
	{ // with `Operation`
		bk, _ := GetBlockByHeight(st, 1)
		bt, _ := GetBlockTransaction(st, bk.Transactions[0])
		bo, _ := GetBlockOperation(st, bt.Operations[0])

		genesisAccount, err := GetBlockAccount(st, bo.Target)
		require.Nil(t, err)

		require.Equal(t, account.Address, genesisAccount.Address)
		require.Equal(t, account.Balance, genesisAccount.Balance)
		require.Equal(t, account.SequenceID, genesisAccount.SequenceID)
	}

	{ // with `Transaction`
		bk, _ := GetBlockByHeight(st, 1)
		bt, _ := GetBlockTransaction(st, bk.Transactions[0])

		genesisAccount, err := GetBlockAccount(st, bt.Source)
		require.Nil(t, err)

		require.Equal(t, account.Address, genesisAccount.Address)
		require.Equal(t, account.Balance, genesisAccount.Balance)
		require.Equal(t, account.SequenceID, genesisAccount.SequenceID)
	}
}
