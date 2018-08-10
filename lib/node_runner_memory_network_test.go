package sebak

import (
	"context"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"github.com/google/uuid"
	"github.com/stellar/go/keypair"
)

func createNetMemoryNetwork() (*sebaknetwork.MemoryNetwork, *sebaknode.LocalNode) {
	mn := sebaknetwork.NewMemoryNetwork()

	kp, _ := keypair.Random()
	localNode, _ := sebaknode.NewLocalNode(kp, mn.Endpoint(), "")

	mn.SetContext(context.WithValue(context.Background(), "localNode", localNode))

	return mn, localNode
}

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
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kp, networkID)

	return
}

func makeTransactionPayment(kpSource *keypair.Full, target string, amount Amount) (tx Transaction) {
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
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func makeTransactionCreateAccount(kpSource *keypair.Full, target string, amount Amount) (tx Transaction) {
	opb := NewOperationBodyCreateAccount(target, Amount(amount))

	op := Operation{
		H: OperationHeader{
			Type: OperationCreateAccount,
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
			Created: sebakcommon.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}
