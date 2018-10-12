package runner

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestGetGenesisAccount(t *testing.T) {
	st := storage.NewTestStorage()

	genesisAccount := block.NewBlockAccount(genesisKP.Address(), common.Amount(1))
	genesisAccount.Save(st)

	commonKP, _ := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)
	commonAccount.Save(st)

	block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)

	fetchedGenesisAccount, err := GetGenesisAccount(st)
	require.Nil(t, err)
	require.Equal(t, genesisAccount.Address, fetchedGenesisAccount.Address)
	require.Equal(t, genesisAccount.Balance, fetchedGenesisAccount.Balance)
	require.Equal(t, genesisAccount.SequenceID, fetchedGenesisAccount.SequenceID)

	fetchedCommonAccount, err := GetCommonAccount(st)
	require.Nil(t, err)
	require.Equal(t, commonAccount.Address, fetchedCommonAccount.Address)
	require.Equal(t, commonAccount.Balance, fetchedCommonAccount.Balance)
	require.Equal(t, commonAccount.SequenceID, fetchedCommonAccount.SequenceID)
}

func TestGetInitialBalance(t *testing.T) {
	st := storage.NewTestStorage()

	initialBalance := common.Amount(99)
	genesisAccount := block.NewBlockAccount(genesisKP.Address(), initialBalance)
	genesisAccount.Save(st)

	commonKP, _ := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)
	commonAccount.Save(st)

	block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)

	fetchedInitialBalance, err := GetGenesisBalance(st)
	require.Nil(t, err)
	require.Equal(t, initialBalance, fetchedInitialBalance)
}
