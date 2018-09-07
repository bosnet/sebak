package sebak

import (
	"boscoin.io/sebak/lib/common"

	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func makeTransaction(kp *keypair.Full) (tx Transaction) {
	var ops []Operation
	ops = append(ops, TestMakeOperation(-1))

	txBody := TransactionBody{
		Source:     kp.Address(),
		Fee:        BaseFee,
		Checkpoint: uuid.New().String(),
		Operations: ops,
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func makeTransactionPayment(kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
	opb := NewOperationBodyPayment(target, amount)

	op := Operation{
		H: OperationHeader{
			Type: OperationPayment,
		},
		B: opb,
	}

	txBody := TransactionBody{
		Source:     kpSource.Address(),
		Fee:        BaseFee,
		Checkpoint: uuid.New().String(),
		Operations: []Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}
