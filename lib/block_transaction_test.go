package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/error"
	"github.com/owlchain/sebak/lib/storage"
)

func TestNewBlockTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)
	a, _ := tx.Serialize()
	bt := NewBlockTransactionFromTransaction(tx, a)

	if bt.Hash != tx.H.Hash {
		t.Error("`BlockTransaction.Hash mismatch`")
		return
	}
	if bt.Checkpoint != tx.B.Checkpoint {
		t.Error("`BlockTransaction.Checkpoint mismatch`")
		return
	}
	if bt.Signature != tx.H.Signature {
		t.Error("`BlockTransaction.Signature mismatch`")
		return
	}
	if bt.Source != tx.B.Source {
		t.Error("`BlockTransaction.Source mismatch`")
		return
	}
	if bt.Fee != tx.B.Fee {
		t.Error("`BlockTransaction.Fee mismatch`")
		return
	}
	if bt.Created != tx.H.Created {
		t.Error("`BlockTransaction.Created mismatch`")
		return
	}
	var opHashes []string
	for _, op := range tx.B.Operations {
		opHashes = append(opHashes, op.MakeHashString())
	}
	for i, opHash := range bt.Operations {
		if opHash != opHashes[i] {
			t.Error("`BlockTransaction.Operations mismatch`")
		}
	}
	if bt.Amount != tx.TotalAmount(true) {
		t.Error("`BlockTransaction.Amount mismatch`")
		return
	}
	if string(bt.Message) != string(a) {
		t.Error("`BlockTransaction.Message mismatch`")
		return
	}
}

func TestBlockTransactionSaveAndGet(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	if err := bt.Save(st); err != nil {
		t.Error(err)
		return
	}

	fetched, err := GetBlockTransaction(st, bt.Hash)
	if err != nil {
		t.Error(err)
		return
	}
	if bt.Hash != fetched.Hash {
		t.Error("mismatch `Hash`")
		return
	}
	if bt.Checkpoint != fetched.Checkpoint {
		t.Error("mismatch `Checkpoint`")
		return
	}
	if bt.Signature != fetched.Signature {
		t.Error("mismatch `Signature`")
		return
	}
	if bt.Source != fetched.Source {
		t.Error("mismatch `Source`")
		return
	}
	if bt.Fee != fetched.Fee {
		t.Error("mismatch `Fee`")
		return
	}
	for i, opHash := range fetched.Operations {
		if opHash != bt.Operations[i] {
			t.Error("mismatch Operation Hashes`")
		}
	}
	if bt.Amount != fetched.Amount {
		t.Error("mismatch `Amount`")
		return
	}
	if bt.Created != fetched.Created {
		t.Error("mismatch `Created`")
		return
	}
	if len(fetched.Confirmed) < 1 {
		t.Error("`Confirmed` missing")
		return
	}
}

func TestBlockTransactionSaveExisting(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	bt.Save(st)

	if exists, err := ExistBlockTransaction(st, bt.Hash); err != nil {
		t.Error(err)
		return
	} else if !exists {
		t.Error("not found")
		return
	}

	if err := bt.Save(st); err == nil {
		t.Error("`ErrorBlockAlreayExists` Errors must be occurred")
		return
	} else if err != sebakerror.ErrorAlreadySaved {
		t.Error("`ErrorAlreadySaved` Errors must be occurred")
		return
	}
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
		if err != nil {
			t.Error(err)
			return
		}
	}

	// create txs from another keypair
	for i := 0; i < numTxs; i++ {
		tx := TestMakeTransactionWithKeypair(networkID, 1, kpAnother)
		a, _ := tx.Serialize()
		bt := NewBlockTransactionFromTransaction(tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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

		if len(saved) != len(createdOrder) {
			t.Error("fetched records insufficient")
			return
		}
		for i, bt := range saved {
			if bt.Hash != createdOrder[i] {
				t.Error("order mismatch")
				return
			}
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

		if len(saved) != len(createdOrder) {
			t.Error("fetched records insufficient")
			return
		}
		reverseCreatedOrder := sebakcommon.ReverseStringSlice(createdOrder)

		for i, bt := range saved {
			if bt.Hash != reverseCreatedOrder[i] {
				t.Error("reverse order mismatch")
				return
			}
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
		if err != nil {
			t.Error(err)
			return
		}
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

	if len(saved) != len(createdOrder) {
		t.Errorf("fetched records insufficient: %d != %d", len(saved), len(createdOrder))
		return
	}
	for i, bt := range saved {
		if bt.Hash != createdOrder[i] {
			t.Error("order mismatch")
			return
		}
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

		if len(saved) != len(createdOrder) {
			t.Error("fetched records insufficient")
			return
		}
		reverseCreatedOrder := sebakcommon.ReverseStringSlice(createdOrder)

		for i, bt := range saved {
			if bt.Hash != reverseCreatedOrder[i] {
				t.Error("reverse order mismatch")
				return
			}
		}
	}
}

func TestBlockTransactionMultipleSave(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	if err := bt.Save(st); err != nil {
		t.Error(err)
		return
	}

	if err := bt.Save(st); err != nil {
		if err != sebakerror.ErrorAlreadySaved {
			t.Errorf("mutiple saving will occur error, 'ErrorAlreadySaved': %v", err)
			return
		}
	}

}

func TestBlockTransactionGetByCheckpoint(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bt := TestMakeNewBlockTransaction(networkID, 1)
	if err := bt.Save(st); err != nil {
		t.Error(err)
		return
	}

	fetched, err := GetBlockTransactionByCheckpoint(st, bt.Checkpoint)
	if err != nil {
		t.Error(err)
		return
	}
	if bt.Hash != fetched.Hash {
		t.Error("mismatch `Hash`")
		return
	}
	if bt.Checkpoint != fetched.Checkpoint {
		t.Error("mismatch `Checkpoint`")
		return
	}
	if bt.Signature != fetched.Signature {
		t.Error("mismatch `Signature`")
		return
	}
	if bt.Source != fetched.Source {
		t.Error("mismatch `Source`")
		return
	}
	if bt.Fee != fetched.Fee {
		t.Error("mismatch `Fee`")
		return
	}
	for i, opHash := range fetched.Operations {
		if opHash != bt.Operations[i] {
			t.Error("mismatch Operation Hashes`")
		}
	}
	if bt.Amount != fetched.Amount {
		t.Error("mismatch `Amount`")
		return
	}
	if bt.Created != fetched.Created {
		t.Error("mismatch `Created`")
		return
	}
	if len(fetched.Confirmed) < 1 {
		t.Error("`Confirmed` missing")
		return
	}
}
