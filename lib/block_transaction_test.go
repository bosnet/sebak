package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"

	"github.com/stretchr/testify/require"
)

func TestNewBlockTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)
	a, _ := tx.Serialize()
	bt := NewBlockTransactionFromTransaction(tx, a)

	require.Equal(t, bt.Hash, tx.H.Hash)
	require.Equal(t, bt.PreviousCheckpoint, tx.B.Checkpoint)
	require.Equal(t, bt.Signature, tx.H.Signature)
	require.Equal(t, bt.Source, tx.B.Source)
	require.Equal(t, bt.Fee, tx.B.Fee)
	require.Equal(t, bt.Created, tx.H.Created)

	var opHashes []string
	for _, op := range tx.B.Operations {
		opHashes = append(opHashes, NewBlockOperationKey(op, tx))
	}
	for i, opHash := range bt.Operations {
		require.Equal(t, opHash, opHashes[i])
	}
	require.Equal(t, bt.Amount, tx.TotalAmount(true))
	require.Equal(t, bt.Message, a)
}

func TestBlockTransactionSaveAndGet(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	err := bt.Save(st)
	require.Nil(t, err)

	fetched, err := GetBlockTransaction(st, bt.Hash)
	require.Nil(t, err)

	require.Equal(t, bt.Hash, fetched.Hash)
	require.Equal(t, bt.PreviousCheckpoint, fetched.PreviousCheckpoint)
	require.Equal(t, bt.Signature, fetched.Signature)
	require.Equal(t, bt.Source, fetched.Source)
	require.Equal(t, bt.Fee, fetched.Fee)
	require.Equal(t, bt.Created, fetched.Created)
	require.Equal(t, bt.Operations, fetched.Operations)
	require.Equal(t, len(fetched.Confirmed) > 0, true)
}

func TestBlockTransactionSaveExisting(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	err := bt.Save(st)
	require.Nil(t, err)

	exists, err := ExistBlockTransaction(st, bt.Hash)
	require.Nil(t, err)
	require.Equal(t, exists, true)

	err = bt.Save(st)
	require.NotNil(t, err)
	require.Equal(t, err, sebakerror.ErrorAlreadySaved)
}

/*
func TestGetSortedBlockTransactionsByCheckpoint(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	// create 30 `BlockOperation`
	var createdOrder []string

	checkpoint := uuid.New().String()
	for i := 0; i < 10; i++ {
		bt := TestMakeNewBlockTransaction(1)
		bt.Checkpoint = checkpoint
		createdOrder = append(createdOrder, bt.Hash)
	}

	var saved []BlockTransaction
	iterFunc, closeFunc := GetBlockTransactionsByCheckpoint(st, checkpoint, false)
	for {
		bo, hasNext := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, bo)
	}
	closeFunc()

	for i, bt := range saved {
		if bt.Hash != createdOrder[i] {
			t.Error("order mismatch")
			break
		}
	}
}
*/

func TestMultipleBlockTransactionSource(t *testing.T) {
	kp, _ := keypair.Random()
	kpAnother, _ := keypair.Random()
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	numTxs := 10

	var txs []Transaction
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother)
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsBySource(st, kp.Address(), false)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, createdOrder[i], bt.Hash)
		}
	}

	{
		// reverse order
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsBySource(st, kp.Address(), true)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		reverseCreatedOrder := sebakcommon.ReverseStringSlice(createdOrder)

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, reverseCreatedOrder[i], bt.Hash)
		}
	}
}

func TestMultipleBlockTransactionConfirmed(t *testing.T) {
	kp, _ := keypair.Random()
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	numTxs := 10

	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		createdOrder = append(createdOrder, tx.GetHash())
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	var saved []BlockTransaction
	iterFunc, closeFunc := GetBlockTransactionsByConfirmed(st, false)
	for {
		bo, hasNext := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, bo)
	}
	closeFunc()

	require.Equal(t, len(saved), len(createdOrder))
	for i, bt := range saved {
		require.Equal(t, createdOrder[i], bt.Hash)
	}

	{
		// reverse order
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByConfirmed(st, true)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		reverseCreatedOrder := sebakcommon.ReverseStringSlice(createdOrder)

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, reverseCreatedOrder[i], bt.Hash)
		}
	}
}

func TestBlockTransactionMultipleSave(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	err := bt.Save(st)
	require.Nil(t, err)

	if err = bt.Save(st); err != nil {
		if err != sebakerror.ErrorAlreadySaved {
			t.Errorf("mutiple saving will occur error, 'ErrorAlreadySaved': %v", err)
			return
		}
	}
}

func TestBlockTransactionGetByCheckpoint(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	err := bt.Save(st)
	require.Nil(t, err)

	fetched, err := GetBlockTransactionByCheckpoint(st, bt.SourceCheckpoint)
	require.Nil(t, err)

	require.Equal(t, bt.Hash, fetched.Hash)
	require.Equal(t, bt.PreviousCheckpoint, fetched.PreviousCheckpoint)
	require.Equal(t, bt.Signature, fetched.Signature)
	require.Equal(t, bt.Source, fetched.Source)
	require.Equal(t, bt.Fee, fetched.Fee)
	require.Equal(t, bt.Created, fetched.Created)
	require.Equal(t, bt.Operations, fetched.Operations)
	require.Equal(t, len(fetched.Confirmed) > 0, true)
}

func TestMultipleBlockTransactionGetByAccount(t *testing.T) {
	kp, _ := keypair.Random()
	kpAnother, _ := keypair.Random()
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	numTxs := 5

	var txs []Transaction
	var createdOrder []string

	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair source but target is this keypair
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())

		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother)
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByAccount(st, kp.Address(), false)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder[i])
		}
	}
}
