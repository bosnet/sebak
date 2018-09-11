package noderunner

import (
	"boscoin.io/sebak/lib/common"

	"boscoin.io/sebak/lib/transaction"
	"github.com/stellar/go/keypair"
)

func makeTransaction(kp *keypair.Full) (tx transaction.Transaction) {
	var ops []transaction.Operation
	ops = append(ops, transaction.TestMakeOperation(-1))

	txBody := transaction.TransactionBody{
		Source:     kp.Address(),
		Fee:        common.BaseFee,
		SequenceID: 0,
		Operations: ops,
	}

	tx = transaction.Transaction{
		T: "transaction",
		H: transaction.TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func makeTransactionPayment(kpSource *keypair.Full, target string, amount common.Amount) (tx transaction.Transaction) {
	opb := transaction.NewOperationBodyPayment(target, amount)

	op := transaction.Operation{
		H: transaction.OperationHeader{
			Type: transaction.OperationPayment,
		},
		B: opb,
	}

	txBody := transaction.TransactionBody{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		SequenceID: 0,
		Operations: []transaction.Operation{op},
	}

	tx = transaction.Transaction{
		T: "transaction",
		H: transaction.TransactionHeader{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}
