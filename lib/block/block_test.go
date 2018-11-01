package block

import (
	"math/rand"
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestBlockConfirmedOrdering(t *testing.T) {
	st := InitTestBlockchain()

	genesis := GetLatestBlock(st)
	inserted := []Block{genesis}
	for i := genesis.Height + 1; i < 10; i++ {
		bk := TestMakeNewBlockWithPrevBlock(inserted[len(inserted)-1], []string{})
		bk.MustSave(st)
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
	st := InitTestBlockchain()

	// save Block, but Height will be shuffled
	maximumHeight := 10
	inserted := make([]Block, maximumHeight+1)

	for i := 2; i <= maximumHeight; i++ {
		bk := TestMakeNewBlock([]string{})
		bk.Height = uint64(i)
		require.NoError(t, bk.Save(st))
		inserted[i] = bk
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for _, i := range r.Perm(maximumHeight) {
		if i == 0 {
			continue
		}
		b, err := GetBlockByHeight(st, uint64(i))
		require.NoError(t, err)
		if i != 1 {
			require.Equal(t, b.Hash, inserted[i].Hash, "Mismatch for", i)
			s, _ := b.Serialize()
			rs, _ := inserted[i].Serialize()
			require.Equal(t, s, rs)
		}
	}
}

// TestMakeGenesisBlock basically tests MakeGenesisBlock can make genesis block,
// and further with genesis block, genesis account can be found.
func TestMakeGenesisBlock(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	genesisKP, _ := keypair.Random()
	balance := common.Amount(100)
	genesisAccount := NewBlockAccount(genesisKP.Address(), balance)
	err := genesisAccount.Save(st)
	require.NoError(t, err)

	commonKP, _ := keypair.Random()
	commonAccount := NewBlockAccount(commonKP.Address(), 0)
	err = commonAccount.Save(st)
	require.NoError(t, err)

	bk, err := MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)
	require.NoError(t, err)
	require.Equal(t, uint64(1), bk.Height)
	require.Equal(t, 1, len(bk.Transactions))
	require.Equal(t, uint64(0), bk.Round)
	require.Equal(t, "", bk.PrevBlockHash)
	require.Equal(t, "", bk.Proposer)
	require.Equal(t, common.GenesisBlockConfirmedTime, bk.Confirmed)

	// transaction
	{
		exists, err := ExistsBlockTransaction(st, bk.Transactions[0])
		require.NoError(t, err)
		require.True(t, exists)
	}

	bt, err := GetBlockTransaction(st, bk.Transactions[0])
	require.NoError(t, err)

	genesisBlockKP := keypair.Master(string(networkID))
	require.Equal(t, genesisAccount.SequenceID, bt.SequenceID)
	require.Equal(t, common.Amount(0), bt.Fee)
	require.Equal(t, 2, len(bt.Operations))
	require.Equal(t, genesisBlockKP.Address(), bt.Source)
	require.Equal(t, bk.Hash, bt.Block)

	// operation
	{
		exists, err := ExistsBlockOperation(st, bt.Operations[0])
		require.NoError(t, err)
		require.True(t, exists)
	}
	bo, err := GetBlockOperation(st, bt.Operations[0])
	require.NoError(t, err)
	require.Equal(t, bt.Hash, bo.TxHash)
	require.Equal(t, operation.TypeCreateAccount, bo.Type)
	require.Equal(t, genesisBlockKP.Address(), bo.Source)

	{
		opb, err := operation.UnmarshalBodyJSON(bo.Type, bo.Body)
		require.NoError(t, err)

		opbp := opb.(operation.Payable)

		require.Equal(t, genesisAccount.Address, opbp.TargetAddress())
		require.Equal(t, genesisAccount.Balance, opbp.GetAmount())
	}
}

func TestMakeGenesisBlockOverride(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	{ // create genesis block
		kp, _ := keypair.Random()
		balance := common.Amount(100)
		account := NewBlockAccount(kp.Address(), balance)
		err := account.Save(st)
		require.NoError(t, err)

		commonKP, _ := keypair.Random()
		commonAccount := NewBlockAccount(commonKP.Address(), 0)
		err = commonAccount.Save(st)
		require.NoError(t, err)

		bk, err := MakeGenesisBlock(st, *account, *commonAccount, networkID)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bk.Height)
	}

	{ // try again to create genesis block
		kp, _ := keypair.Random()
		balance := common.Amount(100)
		account := NewBlockAccount(kp.Address(), balance)
		err := account.Save(st)
		require.NoError(t, err)

		commonKP, _ := keypair.Random()
		commonAccount := NewBlockAccount(commonKP.Address(), 0)
		err = commonAccount.Save(st)
		require.NoError(t, err)

		_, err = MakeGenesisBlock(st, *account, *commonAccount, networkID)
		require.Equal(t, errors.BlockAlreadyExists, err)
	}
}

func TestMakeGenesisBlockFindGenesisAccount(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	// create genesis block
	kp, _ := keypair.Random()
	balance := common.Amount(100)
	account := NewBlockAccount(kp.Address(), balance)
	account.MustSave(st)

	commonKP, _ := keypair.Random()
	commonAccount := NewBlockAccount(commonKP.Address(), 0)
	commonAccount.MustSave(st)

	{
		bk, err := MakeGenesisBlock(st, *account, *commonAccount, networkID)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bk.Height)
	}

	// find genesis account
	{ // with `Operation`
		bk := GetGenesis(st)
		bt, _ := GetBlockTransaction(st, bk.Transactions[0])
		bo, _ := GetBlockOperation(st, bt.Operations[0])

		opb, err := operation.UnmarshalBodyJSON(bo.Type, bo.Body)
		require.NoError(t, err)

		opbp := opb.(operation.Payable)

		genesisAccount, err := GetBlockAccount(st, opbp.TargetAddress())
		require.NoError(t, err)

		require.Equal(t, account.Address, genesisAccount.Address)
		require.Equal(t, account.Balance, genesisAccount.Balance)
		require.Equal(t, account.SequenceID, genesisAccount.SequenceID)
	}
}

func TestMakeGenesisBlockFindCommonAccount(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()

	// create genesis block
	kp, _ := keypair.Random()
	balance := common.Amount(100)
	genesisAccount := NewBlockAccount(kp.Address(), balance)
	genesisAccount.MustSave(st)

	commonKP, _ := keypair.Random()
	commonAccount := NewBlockAccount(commonKP.Address(), 0)
	commonAccount.MustSave(st)

	{
		bk, err := MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)
		require.NoError(t, err)
		require.Equal(t, uint64(1), bk.Height)
	}

	// find common account
	{ // with `Operation`
		bk := GetGenesis(st)
		bt, _ := GetBlockTransaction(st, bk.Transactions[0])
		bo, _ := GetBlockOperation(st, bt.Operations[1])

		opb, err := operation.UnmarshalBodyJSON(bo.Type, bo.Body)
		require.NoError(t, err)

		opbp := opb.(operation.Payable)

		ac, err := GetBlockAccount(st, opbp.TargetAddress())
		require.NoError(t, err)

		require.Equal(t, commonAccount.Address, ac.Address)
		require.Equal(t, commonAccount.Balance, ac.Balance)
		require.Equal(t, commonAccount.SequenceID, ac.SequenceID)
	}
}
