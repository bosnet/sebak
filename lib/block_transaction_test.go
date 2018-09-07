package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

func TestNewBlockTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)
	a, _ := tx.Serialize()
	block := testMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)

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
	require.Equal(t, err, errors.ErrorAlreadySaved)
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
	var txHashes []string
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())
		txHashes = append(txHashes, tx.GetHash())

	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair
	txs = []Transaction{}
	txHashes = []string{}
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block = testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
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

		reverseCreatedOrder := common.ReverseStringSlice(createdOrder)

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

	var txs []Transaction
	var txHashes []string
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		createdOrder = append(createdOrder, tx.GetHash())
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
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

		reverseCreatedOrder := common.ReverseStringSlice(createdOrder)

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
		if err != errors.ErrorAlreadySaved {
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
	var txHashes []string
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())
		txHashes = append(txHashes, tx.GetHash())
	}

	block := testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair source but target is this keypair
	txs = []Transaction{}
	txHashes = []string{}
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())
		txHashes = append(txHashes, tx.GetHash())
	}

	block = testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
		err := bt.Save(st)
		require.Nil(t, err)
	}

	// create txs from another keypair
	txs = []Transaction{}
	txHashes = []string{}
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block = testMakeNewBlock(txHashes)
	for _, tx := range txs {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block.Hash, tx, a)
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

func TestMultipleBlockTransactionGetByBlock(t *testing.T) {
	kp, _ := keypair.Random()
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	numTxs := 5

	var txs0 []Transaction
	var txHashes0 []string
	var createdOrder0 []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs0 = append(txs0, tx)
		createdOrder0 = append(createdOrder0, tx.GetHash())
		txHashes0 = append(txHashes0, tx.GetHash())
	}

	block0 := testMakeNewBlock(txHashes0)
	for _, tx := range txs0 {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block0.Hash, tx, a)
		require.Nil(t, bt.Save(st))
	}

	var txs1 []Transaction
	var txHashes1 []string
	var createdOrder1 []string
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kp)
		txs1 = append(txs1, tx)
		createdOrder1 = append(createdOrder1, tx.GetHash())
		txHashes1 = append(txHashes1, tx.GetHash())
	}

	block1 := testMakeNewBlock(txHashes1)
	for _, tx := range txs1 {
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(block1.Hash, tx, a)
		require.Nil(t, bt.Save(st))
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByBlock(st, block0.Hash, false)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder0), "fetched records insufficient")
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder0[i], "order mismatch")
		}
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByBlock(st, block1.Hash, false)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder1), "fetched records insufficient")
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder1[i], "order mismatch")
		}
	}
}
