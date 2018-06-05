package sebak

import (
	"math/rand"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"

	"github.com/owlchain/sebak/lib/common"
)

func testMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := Amount(2000)
	hashed := sebakcommon.MustMakeObjectHash("")
	checkpoint := base58.Encode(hashed)

	return NewBlockAccount(address, balance, checkpoint)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := TestMakeTransaction(networkID, n)

	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(tx, a)
}

func TestMakeOperationBodyPayment(amount int) OperationBodyPayment {
	kp, _ := keypair.Random()

	for amount < 0 {
		amount = rand.Intn(5000)
	}

	return OperationBodyPayment{
		Target: kp.Address(),
		Amount: Amount(amount),
	}
}

func TestMakeOperation(amount int) Operation {
	opb := TestMakeOperationBodyPayment(amount)

	op := Operation{
		H: OperationHeader{
			Type: OperationPayment,
		},
		B: opb,
	}

	return op
}

func TestMakeTransaction(networkID []byte, n int) (kp *keypair.Full, tx Transaction) {
	kp, _ = keypair.Random()

	var ops []Operation
	for i := 0; i < n; i++ {
		ops = append(ops, TestMakeOperation(-1))
	}

	txBody := TransactionBody{
		Source:     kp.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func TestMakeTransactionWithKeypair(networkID []byte, n int, kp *keypair.Full) (tx Transaction) {
	var ops []Operation
	for i := 0; i < n; i++ {
		ops = append(ops, TestMakeOperation(-1))
	}
	tx, _ = NewTransaction(kp.Address(), uuid.New().String(), ops...)
	tx.Sign(kp, networkID)

	return
}
