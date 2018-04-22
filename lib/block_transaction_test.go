package sebak

import (
	"testing"

	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/storage"
)

func TestNewBlockTransaction(t *testing.T) {
	tx := makeTransaction(1)
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
		opHashes = append(opHashes, op.H.Hash)
	}
	for i, opHash := range bt.Operations {
		if opHash != opHashes[i] {
			t.Error("`BlockTransaction.Operations mismatch`")
		}
	}
	if bt.Amount != tx.GetTotalAmount(true) {
		t.Error("`BlockTransaction.Amount mismatch`")
		return
	}
	if string(bt.Message) != string(a) {
		t.Error("`BlockTransaction.Message mismatch`")
		return
	}
}

func TestBlockTransactionSaveAndGet(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	bt := makeNewBlockTransaction(1)
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
	st, _ := storage.NewTestMemoryLevelDBBackend()

	bt := makeNewBlockTransaction(1)
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
	} else if err != ErrorBlockAlreayExists {
		t.Error("`ErrorBlockAlreayExists` Errors must be occurred")
		return
	}
}

func TestGetSortedBlockTransactionsByCheckpoint(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	// create 30 `BlockOperation`
	var createdOrder []string

	checkpoint := uuid.New().String()
	for i := 0; i < 10; i++ {
		bt := makeNewBlockTransaction(1)
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
