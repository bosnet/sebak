package transaction

import (
	"math/rand"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/transaction/operation"
)

func TestMakeTransaction(networkID []byte, n int) (kp *keypair.Full, tx Transaction) {
	kp = keypair.Random()

	var ops []operation.Operation
	for i := 0; i < n; i++ {
		ops = append(ops, operation.MakeTestPayment(-1))
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
	} else {
		k := keypair.Random()
		targetAddr = k.Address()
	}

	for i := 0; i < n; i++ {
		ops = append(ops, operation.MakeTestPaymentTo(-1, targetAddr))
	}

	tx, _ = NewTransaction(
		srcKp.Address(),
		0,
		ops...,
	)
	tx.Sign(srcKp, networkID)

	return
}

func MakeTransactionCreateAccount(networkID []byte, kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
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
		H: Header{
			Version: common.TransactionVersionV1,
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionCreateFrozenAccount(networkID []byte, kpSource *keypair.Full, target string, amount common.Amount, linkedAccount string) (tx Transaction) {
	opb := operation.NewFreezing(target, common.Amount(amount), linkedAccount)

	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeFreezing,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.Amount(0),
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionPayment(networkID []byte, kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
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
		H: Header{
			Version: common.TransactionVersionV1,
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}
	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionUnfreezingRequest(networkID []byte, kpSource *keypair.Full) (tx Transaction) {
	opb := operation.NewUnfreezeRequest()
	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeUnfreezingRequest,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.Amount(0),
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	tx.Sign(kpSource, networkID)

	return
}

func MakeTransactionUnfreezing(networkID []byte, kpSource *keypair.Full, target string, amount common.Amount) (tx Transaction) {
	opb := operation.NewUnfreezing(target, common.Amount(amount))
	op := operation.Operation{
		H: operation.Header{
			Type: operation.TypeUnfreezing,
		},
		B: opb,
	}

	txBody := Body{
		Source:     kpSource.Address(),
		Fee:        common.Amount(0),
		Operations: []operation.Operation{op},
	}

	tx = Transaction{
		H: Header{
			Created: common.NowISO8601(),
			Hash:    txBody.MakeHashString(),
		},
		B: txBody,
	}

	tx.Sign(kpSource, networkID)

	return
}
