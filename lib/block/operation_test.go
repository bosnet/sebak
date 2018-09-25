package block

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"

	"boscoin.io/sebak/lib/transaction"
	"github.com/stretchr/testify/require"
)

func TestNewBlockOperationFromOperation(t *testing.T) {
	_, tx := transaction.TestMakeTransaction(networkID, 1)

	op := tx.B.Operations[0]
	bo, err := NewBlockOperationFromOperation(op, tx, 0)
	require.Nil(t, err)

	require.Equal(t, bo.Type, op.H.Type)
	require.Equal(t, bo.TxHash, tx.H.Hash)
	require.Equal(t, bo.Source, tx.B.Source)
	encoded, err := op.B.Serialize()
	require.Nil(t, err)
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
	require.Nil(t, err)

	require.Equal(t, bo.Type, fetched.Type)
	require.Equal(t, bo.Hash, fetched.Hash)
	require.Equal(t, bo.Source, fetched.Source)
	require.Equal(t, bo.Body, fetched.Body)
}

func TestBlockOperationSaveExisting(t *testing.T) {
	st := storage.NewTestStorage()

	bos := TestMakeNewBlockOperation(networkID, 1)
	bo := bos[0]
	bo.Save(st)

	exists, err := ExistsBlockOperation(st, bos[0].Hash)
	require.Nil(t, err)
	require.Equal(t, exists, true)

	err = bo.Save(st)
	require.NotNil(t, err, "An error should have been returned")
	require.Equal(t, err, errors.ErrorAlreadySaved)
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
			bo.Save(st)

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

func TestBlockOperationSaveByTransacton(t *testing.T) {
	st := storage.NewTestStorage()

	_, tx := transaction.TestMakeTransaction(networkID, 10)
	block := TestMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.Confirmed, tx, common.MustJSONMarshal(tx))
	err := bt.Save(st)
	require.Nil(t, err)

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
		require.Nil(t, err)
		require.Equal(t, bo.Body, encoded)
	}
}
