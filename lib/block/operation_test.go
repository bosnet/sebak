package block

import (
	"testing"

	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/transaction"
)

func TestNewBlockOperationFromOperation(t *testing.T) {
	_, tx := transaction.TestMakeTransaction(networkID, 1)

	op := tx.B.Operations[0]
	bo, err := NewBlockOperationFromOperation(op, tx, 0)
	require.NoError(t, err)

	require.Equal(t, bo.Type, op.H.Type)
	require.Equal(t, bo.TxHash, tx.H.Hash)
	require.Equal(t, bo.Source, tx.B.Source)
	encoded, err := op.B.Serialize()
	require.NoError(t, err)
	require.Equal(t, bo.Body, encoded)
}

func TestBlockOperationSaveAndGet(t *testing.T) {
	st := storage.NewTestStorage()

	bos := TestMakeNewBlockOperation(networkID, 1)
	if err := bos[0].Save(st); err != nil {
		t.Error(err)
		return
	}

	bo := bos[0]
	fetched, err := GetBlockOperation(st, bo.Hash)
	require.NoError(t, err)

	require.Equal(t, bo.Type, fetched.Type)
	require.Equal(t, bo.Hash, fetched.Hash)
	require.Equal(t, bo.Source, fetched.Source)
	require.Equal(t, bo.Body, fetched.Body)
}

func TestBlockOperationSaveExisting(t *testing.T) {
	st := storage.NewTestStorage()

	bos := TestMakeNewBlockOperation(networkID, 1)
	bo := bos[0]
	bo.MustSave(st)

	exists, err := ExistsBlockOperation(st, bos[0].Hash)
	require.NoError(t, err)
	require.Equal(t, exists, true)

	err = bo.Save(st)
	require.NotNil(t, err, "An error should have been returned")
	require.Equal(t, err, errors.AlreadySaved)
}

func TestGetSortedBlockOperationsByTxHash(t *testing.T) {
	st := storage.NewTestStorage()

	// create 30 `BlockOperation`
	var txHashes []string
	createdOrder := map[string][]string{}
	for _ = range [3]int{0, 0, 0} {
		bos := TestMakeNewBlockOperation(networkID, 10)
		txHashes = append(txHashes, bos[0].TxHash)

		for _, bo := range bos {
			bo.MustSave(st)

			createdOrder[bo.TxHash] = append(createdOrder[bo.TxHash], bo.Hash)
		}
	}

	for _, txHash := range txHashes {
		var saved []BlockOperation
		iterFunc, closeFunc := GetBlockOperationsByTxHash(st, txHash, nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		for i, bo := range saved {
			require.Equal(t, bo.Hash, createdOrder[bo.TxHash][i])
		}
	}
}

func TestBlockOperationSaveByTransaction(t *testing.T) {
	st := InitTestBlockchain()

	_, tx := transaction.TestMakeTransaction(networkID, 10)
	block := TestMakeNewBlockWithPrevBlock(GetLatestBlock(st), []string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.Confirmed, tx)
	err := bt.Save(st)
	require.NoError(t, err)

	var saved []BlockOperation
	iterFunc, closeFunc := GetBlockOperationsByTxHash(st, tx.GetHash(), nil)
	for {
		bo, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, bo)
	}
	closeFunc()

	for i, bo := range saved {
		op := tx.B.Operations[i]
		require.Equal(t, bo.Type, op.H.Type)
		require.Equal(t, bo.TxHash, tx.H.Hash)
		require.Equal(t, bo.Source, tx.B.Source)
		encoded, err := op.B.Serialize()
		require.NoError(t, err)
		require.Equal(t, bo.Body, encoded)
	}
}
