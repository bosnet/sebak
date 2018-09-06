package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestNewBlockOperationFromOperation(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)

	op := tx.B.Operations[0]
	bo := NewBlockOperationFromOperation(op, tx)

	require.Equal(t, bo.Type, op.H.Type)
	require.Equal(t, bo.TxHash, tx.H.Hash)
	require.Equal(t, bo.Source, tx.B.Source)
	require.Equal(t, bo.Target, op.B.TargetAddress())
	require.Equal(t, bo.Amount, op.B.GetAmount())
}

func TestBlockOperationSaveAndGet(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

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
	require.Equal(t, bo.Target, fetched.Target)
	require.Equal(t, bo.Amount, fetched.Amount)
}

func TestBlockOperationSaveExisting(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bos := TestMakeNewBlockOperation(networkID, 1)
	bo := bos[0]
	bo.Save(st)

	exists, err := ExistBlockOperation(st, bos[0].Hash)
	require.Nil(t, err)
	require.Equal(t, exists, true)

	err = bo.Save(st)
	require.NotNil(t, err, "An error should have been returned")
	require.Equal(t, err, errors.ErrorAlreadySaved)
}

func TestGetSortedBlockOperationsByTxHash(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

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
		iterFunc, closeFunc := GetBlockOperationsByTxHash(st, txHash, false)
		for {
			bo, hasNext := iterFunc()
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
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	_, tx := TestMakeTransaction(networkID, 10)
	block := testMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, tx, sebakcommon.MustJSONMarshal(tx))
	err := bt.Save(st)
	require.Nil(t, err)

	var saved []BlockOperation
	iterFunc, closeFunc := GetBlockOperationsByTxHash(st, tx.GetHash(), false)
	for {
		bo, hasNext := iterFunc()
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
		require.Equal(t, bo.Target, op.B.TargetAddress())
		require.Equal(t, bo.Amount, op.B.GetAmount())
	}
}

func TestBlockOperationGetSortedByCheckpoint(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	_, tx := TestMakeTransaction(networkID, 10)
	block := testMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, tx, sebakcommon.MustJSONMarshal(tx))
	err := bt.Save(st)
	require.Nil(t, err)

	{
		_, txAnother := TestMakeTransaction(networkID, 10)
		block := testMakeNewBlock([]string{txAnother.GetHash()})
		btAnother := NewBlockTransactionFromTransaction(block.Hash, txAnother, sebakcommon.MustJSONMarshal(tx))
		err2 := btAnother.Save(st)
		require.Nil(t, err2)
	}

	var saved []BlockOperation
	iterFunc, closeFunc := GetBlockOperationsByCheckpoint(st, tx.B.Checkpoint, false)
	for {
		bo, hasNext := iterFunc()
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
		require.Equal(t, bo.Target, op.B.TargetAddress())
		require.Equal(t, bo.Amount, op.B.GetAmount())
	}
}
