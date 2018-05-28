package sebak

import (
	"testing"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
)

func TestNewBlockOperationFromOperation(t *testing.T) {
	_, tx := TestMakeTransaction(1)

	op := tx.B.Operations[0]
	bo := NewBlockOperationFromOperation(op, tx)

	if bo.Type != op.H.Type {
		t.Error("mismatch `Type`")
		return
	}
	if bo.TxHash != tx.H.Hash {
		t.Error("mismatch `TxHash`")
		return
	}
	if bo.Source != tx.B.Source {
		t.Error("mismatch `Source`")
		return
	}
	if bo.Target != op.B.TargetAddress() {
		t.Error("mismatch `Target`")
		return
	}
	if bo.Amount != op.B.GetAmount() {
		t.Error("mismatch `Type`")
		return
	}
}

func TestBlockOperationSaveAndGet(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bos := TestMakeNewBlockOperation(1)
	if err := bos[0].Save(st); err != nil {
		t.Error(err)
		return
	}

	bo := bos[0]
	fetched, err := GetBlockOperation(st, bo.Hash)
	if err != nil {
		t.Error(err)
		return
	}

	if bo.Hash != fetched.Hash {
		t.Error("mismatch `Hash`")
		return
	}
	if bo.Type != fetched.Type {
		t.Error("mismatch `Type`")
		return
	}
	if bo.TxHash != fetched.TxHash {
		t.Error("mismatch `TxHash`")
		return
	}
	if bo.Source != fetched.Source {
		t.Error("mismatch `Source`")
		return
	}
	if bo.Target != fetched.Target {
		t.Error("mismatch `Target`")
		return
	}
	if bo.Amount != fetched.Amount {
		t.Error("mismatch `Amount`")
		return
	}
}

func TestBlockOperationSaveExisting(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	bos := TestMakeNewBlockOperation(1)
	bo := bos[0]
	bo.Save(st)

	if exists, err := ExistBlockOperation(st, bos[0].Hash); err != nil {
		t.Error(err)
		return
	} else if !exists {
		t.Error("not found")
		return
	}

	if err := bo.Save(st); err == nil {
		t.Error("`ErrorBlockAlreayExists` Errors must be occurred")
		return
	} else if err != sebakerror.ErrorAlreadySaved {
		t.Error("`ErrorAlreadySaved` Errors must be occurred")
		return
	}
}

func TestGetSortedBlockOperationsByTxHash(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	// create 30 `BlockOperation`
	var txHashes []string
	createdOrder := map[string][]string{}
	for _ = range [3]int{0, 0, 0} {
		bos := TestMakeNewBlockOperation(10)
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
			if bo.Hash != createdOrder[bo.TxHash][i] {
				t.Error("order mismatch")
				break
			}
		}
	}
}

func TestBlockOperationSaveByTransacton(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	_, tx := TestMakeTransaction(10)
	bt := NewBlockTransactionFromTransaction(tx, sebakcommon.MustJSONMarshal(tx))
	if err := bt.Save(st); err != nil {
		t.Error(err)
		return
	}

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
		if bo.Type != op.H.Type {
			t.Error("mismatch `Type`")
			return
		}
		if bo.TxHash != tx.H.Hash {
			t.Error("mismatch `TxHash`")
			return
		}
		if bo.Source != tx.B.Source {
			t.Error("mismatch `Source`")
			return
		}
		if bo.Target != op.B.TargetAddress() {
			t.Error("mismatch `Target`")
			return
		}
		if bo.Amount != op.B.GetAmount() {
			t.Error("mismatch `Type`")
			return
		}
	}
}

func TestBlockOperationGetSortedByCheckpoint(t *testing.T) {
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

	_, tx := TestMakeTransaction(10)
	bt := NewBlockTransactionFromTransaction(tx, sebakcommon.MustJSONMarshal(tx))
	if err := bt.Save(st); err != nil {
		t.Error(err)
		return
	}

	{
		_, txAnother := TestMakeTransaction(10)
		btAnother := NewBlockTransactionFromTransaction(txAnother, sebakcommon.MustJSONMarshal(tx))
		if err := btAnother.Save(st); err != nil {
			t.Error(err)
			return
		}
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
		if bo.Type != op.H.Type {
			t.Error("mismatch `Type`")
			return
		}
		if bo.TxHash != tx.H.Hash {
			t.Error("mismatch `TxHash`")
			return
		}
		if bo.Source != tx.B.Source {
			t.Error("mismatch `Source`")
			return
		}
		if bo.Target != op.B.TargetAddress() {
			t.Error("mismatch `Target`")
			return
		}
		if bo.Amount != op.B.GetAmount() {
			t.Error("mismatch `Type`")
			return
		}
	}
}
