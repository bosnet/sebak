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

func makeNewBlockOperation(n int) (bos []BlockOperation) {
	tx := MakeTransaction(n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func makeNewBlockTransaction(n int) BlockTransaction {
	tx := MakeTransaction(n)

	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(tx, a)
}

func makeOperationBodyPayment() OperationBodyPayment {
	kp, _ := keypair.Random()

	var amount int
	for amount < 1 {
		amount = rand.Intn(5000)
	}

	return OperationBodyPayment{
		Target: kp.Address(),
		Amount: Amount(amount),
	}
}

func makeOperation() Operation {
	opb := makeOperationBodyPayment()

	op := Operation{
		H: OperationHeader{
			Hash: opb.MakeHashString(),
			Type: OperationPayment,
		},
		B: opb,
	}

	return op
}

func MakeTransaction(n int) (tx Transaction) {
	kpSource, _ := keypair.Random()

	var ops []Operation
	for i := 0; i < n; i++ {
		ops = append(ops, makeOperation())
	}

	txBody := TransactionBody{
		Source:     kpSource.Address(),
		Fee:        Amount(BaseFee),
		Checkpoint: uuid.New().String(),
		Operations: ops,
	}

	tx = Transaction{
		H: TransactionHeader{
			Created: util.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource)

	return
}
