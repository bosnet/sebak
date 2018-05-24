package sebak

import (
	"fmt"
	"math/rand"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

func makeBlockAccount() *BlockAccount {
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := 2000
	hashed := util.MustMakeObjectHash("")
	checkpoint := base58.Encode(hashed)

	return NewBlockAccount(address, fmt.Sprintf("%d", balance), checkpoint)
}

func MakeNewBlockOperation(n int) (bos []BlockOperation) {
	_, tx := MakeTransactions(n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func MakeNewBlockTransaction(n int) BlockTransaction {
	_, tx := MakeTransactions(n)

	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(tx, a)
}

func MakeOperationBodyPayment(amount int) OperationBodyPayment {
	kp, _ := keypair.Random()

	for amount < 0 {
		amount = rand.Intn(5000)
	}

	return OperationBodyPayment{
		Target: kp.Address(),
		Amount: Amount(amount),
	}
}

func MakeOperation(amount int) Operation {
	opb := MakeOperationBodyPayment(amount)

	op := Operation{
		H: OperationHeader{
			Type: OperationPayment,
		},
		B: opb,
	}

	return op
}

func MakeTransactions(n int) (kp *keypair.Full, tx Transaction) {
	kp, _ = keypair.Random()

	var ops []Operation
	for i := 0; i < n; i++ {
		ops = append(ops, MakeOperation(-1))
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
			Created: util.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp)

	return
}
