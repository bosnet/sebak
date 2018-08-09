package sebak

import (
	"fmt"
	"math/rand"

	"boscoin.io/sebak/lib/common"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func testMakeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := Amount(2000)
	checkpoint := TestGenerateNewCheckpoint()

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

func TestMakeOperationBodyPayment(amount int, addressList ...string) OperationBodyPayment {
	var address string
	if len(addressList) > 0 {
		address = addressList[0]
	} else {
		kp, _ := keypair.Random()
		address = kp.Address()
	}

	for amount < 0 {
		amount = rand.Intn(5000)
	}

	return OperationBodyPayment{
		Target: address,
		Amount: Amount(amount),
	}
}

func TestMakeOperation(amount int, addressList ...string) Operation {
	opb := TestMakeOperationBodyPayment(amount, addressList...)

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
		Fee:        BaseFee,
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

func TestGenerateNewCheckpoint() string {
	return uuid.New().String()
}

func TestMakeTransactionWithKeypair(networkID []byte, n int, srcKp *keypair.Full, targetKps ...*keypair.Full) (tx Transaction) {
	var ops []Operation
	var targetAddr string

	if len(targetKps) > 0 {
		targetAddr = targetKps[0].Address()
	}

	for i := 0; i < n; i++ {
		ops = append(ops, TestMakeOperation(-1, targetAddr))
	}

	tx, _ = NewTransaction(
		srcKp.Address(),
		fmt.Sprintf("%s-%s", TestGenerateNewCheckpoint(), TestGenerateNewCheckpoint()),
		ops...,
	)
	tx.Sign(srcKp, networkID)

	return
}
