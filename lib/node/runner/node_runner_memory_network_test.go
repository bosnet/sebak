package runner

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

func makeTransaction(kp *keypair.Full) (tx transaction.Transaction) {
	var ops []operation.Operation
	ops = append(ops, operation.MakeTestPayment(-1))

	txBody := transaction.Body{
		Source:     kp.Address(),
		Fee:        common.BaseFee,
		SequenceID: 0,
		Operations: ops,
	}

	tx = transaction.Transaction{
		H: transaction.Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func makeTransactionPayment(kpSource *keypair.Full, target string, amount common.Amount) (tx transaction.Transaction) {
	opb := operation.NewPayment(target, amount)

	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypePayment,
		},
		B: opb,
	}

	txBody := transaction.Body{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		SequenceID: 0,
		Operations: []operation.Operation{op},
	}

	tx = transaction.Transaction{
		H: transaction.Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}
