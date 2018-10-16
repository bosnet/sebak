package runner

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestGetGenesisAccount(t *testing.T) {
	st := storage.NewTestStorage()

	genesisAccount := block.NewBlockAccount(block.GenesisKP.Address(), common.Amount(1))
	genesisAccount.MustSave(st)

	commonAccount := block.NewBlockAccount(block.CommonKP.Address(), 0)
	commonAccount.MustSave(st)

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
	genesisAccount := block.NewBlockAccount(block.GenesisKP.Address(), initialBalance)
	genesisAccount.MustSave(st)

	commonAccount := block.NewBlockAccount(block.CommonKP.Address(), 0)
	commonAccount.MustSave(st)

	block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)

	fetchedInitialBalance, err := GetGenesisBalance(st)
	require.Nil(t, err)
	require.Equal(t, initialBalance, fetchedInitialBalance)
}
