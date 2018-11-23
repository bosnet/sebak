package block

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestNewBlockOperationFromOperation(t *testing.T) {
	conf := common.NewTestConfig()
	_, tx := transaction.TestMakeTransaction(conf.NetworkID, 1)

	var index int = 0
	var txIndex uint64 = 1

	op := tx.B.Operations[0]
	bo, err := NewBlockOperationFromOperation(op, tx, 0, txIndex, index)
	require.NoError(t, err)

	require.Equal(t, bo.Type, op.H.Type)
	require.Equal(t, bo.TxHash, tx.H.Hash)
	require.Equal(t, bo.Source, tx.B.Source)
	encoded := common.MustMarshalJSON(op.B)
	require.Equal(t, bo.Body, encoded)
	require.Equal(t, bo.Index, index)
	require.Equal(t, bo.TxIndex, txIndex)
}

func TestBlockOperationSaveAndGet(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	bos := TestMakeNewBlockOperation(conf.NetworkID, 1, 0)
	bo := bos[0]
	bos[0].MustSave(st)

	fetched, err := GetBlockOperation(st, bo.Hash)
	require.NoError(t, err)

	require.Equal(t, bo.Type, fetched.Type)
	require.Equal(t, bo.Hash, fetched.Hash)
	require.Equal(t, bo.Source, fetched.Source)
	require.Equal(t, bo.Body, fetched.Body)
	require.Equal(t, bo.Index, uint64(0))
}

func TestBlockOperationSaveExisting(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	bos := TestMakeNewBlockOperation(conf.NetworkID, 1, 0)
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
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	// create 30 `BlockOperation`
	var txHashes []string
	createdOrder := map[string][]string{}
	for _, i := range [3]uint64{0, 1, 2} {
		bos := TestMakeNewBlockOperation(conf.NetworkID, 10, i)
		txHashes = append(txHashes, bos[0].TxHash)

		for _, bo := range bos {
			bo.MustSave(st)

			createdOrder[bo.TxHash] = append(createdOrder[bo.TxHash], bo.Hash)
		}
	}

	for _, txHash := range txHashes {
		var saved []BlockOperation
		iterFunc, closeFunc := GetBlockOperationsByTx(st, txHash, nil)
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
			require.Equal(t, bo.Index, uint64(i))
		}
	}
}

func TestBlockOperationSaveByTransaction(t *testing.T) {
	conf := common.NewTestConfig()
	st := InitTestBlockchain()

	_, tx := transaction.TestMakeTransaction(conf.NetworkID, 10)
	block := TestMakeNewBlockWithPrevBlock(GetLatestBlock(st), []string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, 0)
	err := bt.Save(st)
	require.NoError(t, err)

	var saved []BlockOperation
	iterFunc, closeFunc := GetBlockOperationsByTx(st, tx.GetHash(), nil)
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
		encoded := common.MustMarshalJSON(op.B)
		require.Equal(t, bo.Body, encoded)
	}
}

func TestBlockOperationsByBlockHeight(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	heights := []uint64{1, 2, 3}
	created := map[uint64][]string{}
	for _, height := range heights {
		bos := TestMakeNewBlockOperation(conf.NetworkID, 10, 0)

		for _, bo := range bos {
			bo.Height = height
			bo.MustSave(st)
			created[height] = append(created[height], bo.Hash)
		}

	}

	var saved []BlockOperation
	for _, height := range heights {
		var saved []BlockOperation
		iterFunc, closeFunc := GetBlockOperationsByBlockHeight(st, height, nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			saved = append(saved, bo)
		}
		closeFunc()

		for i, bo := range saved {
			require.Equal(t, bo.Hash, created[height][i])
		}
	}

	for _, bo := range saved {
		println(bo.Height, bo.Hash)

	}

}
