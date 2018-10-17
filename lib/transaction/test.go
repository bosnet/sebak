package transaction

import (
	"math/rand"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction/operation"
)

var (
	networkID []byte = []byte("sebak-test-network")
	kp        *keypair.Full
)

func init() {
	kp, _ = keypair.Random()
}

func TestMakeTransaction(networkID []byte, n int) (kp *keypair.Full, tx Transaction) {
	kp, _ = keypair.Random()

	var ops []operation.Operation
	for i := 0; i < n; i++ {
		ops = append(ops, operation.TestMakeOperation(-1))
	}

	tx, _ = NewTransaction(kp.Address(), 0, ops...)
	tx.Sign(kp, networkID)

	return
}

func TestMakeTransactionWithKeypair(networkID []byte, n int, srcKp *keypair.Full, targetKps ...*keypair.Full) (tx Transaction) {
	var ops []operation.Operation
	var targetAddr string

	if len(targetKps) > 0 {
		targetAddr = targetKps[0].Address()
	}

	for i := 0; i < n; i++ {
		ops = append(ops, operation.TestMakeOperation(-1, targetAddr))
	}

	tx, _ = NewTransaction(
		srcKp.Address(),
		0,
		ops...,
	)
	tx.Sign(srcKp, networkID)

	return
}

func MakeTransactionCreateAccount(kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
	opb := operation.NewCreateAccount(target, common.Amount(amount), "")

	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeCreateAccount,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		SequenceID: rand.Uint64(),
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionCreateFrozenAccount(kpSource *keypair.Full, target string, amount common.Amount, linkedAccount string) (tx Transaction) {
	opb := operation.NewCreateAccount(target, common.Amount(amount), linkedAccount)

	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeCreateAccount,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionUnfreezingRequest(kpSource *keypair.Full) (tx Transaction) {
	opb := operation.NewUnfreezeRequest()
	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeUnfreezingRequest,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionUnfreezing(kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
	opb := operation.NewPayment(target, common.Amount(amount))
	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypePayment,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.BaseFee,
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		T: "transaction",
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	tx.Sign(kpSource, networkID)

	return
}
