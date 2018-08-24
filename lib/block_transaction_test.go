package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

func TestNewBlockTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)
	a, _ := tx.Serialize()
	block := testMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block, tx, a)

	if bt.Hash != tx.H.Hash {
		t.Error("`BlockTransaction.Hash mismatch`")
		return
	}
	if bt.PreviousCheckpoint != tx.B.Checkpoint {
		t.Error("`BlockTransaction.PreviousCheckpoint mismatch`")
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
		opHashes = append(opHashes, NewBlockOperationKey(op, tx))
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
	if bt.PreviousCheckpoint != fetched.PreviousCheckpoint {
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
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

	fetched, err := GetBlockTransactionByCheckpoint(st, bt.SourceCheckpoint)
	if err != nil {
		t.Error(err)
		return
	}
	if bt.Hash != fetched.Hash {
		t.Error("mismatch `Hash`")
		return
	}
	if bt.PreviousCheckpoint != fetched.PreviousCheckpoint {
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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
		bt := NewBlockTransactionFromTransaction(block, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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

		if len(saved) != len(createdOrder) {
			t.Error("fetched records insufficient")
		}
		for i, bt := range saved {
			if bt.Hash != createdOrder[i] {
				t.Error("order mismatch")
				return
			}
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
		bt := NewBlockTransactionFromTransaction(block0, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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
		bt := NewBlockTransactionFromTransaction(block1, tx, a)
		err := bt.Save(st)
		if err != nil {
			t.Error(err)
			return
		}
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

		if len(saved) != len(createdOrder0) {
			t.Error("fetched records insufficient")
		}
		for i, bt := range saved {
			if bt.Hash != createdOrder0[i] {
				t.Error("order mismatch")
				return
			}
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

		if len(saved) != len(createdOrder1) {
			t.Error("fetched records insufficient")
		}
		for i, bt := range saved {
			if bt.Hash != createdOrder1[i] {
				t.Error("order mismatch")
				return
			}
		}

	}
}
