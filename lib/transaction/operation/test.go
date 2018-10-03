package operation

import (
	"math/rand"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
)

var (
	networkID []byte = []byte("sebak-test-network")
	kp        *keypair.Full
)

func init() {
	kp, _ = keypair.Random()
}

func TestMakeOperationBodyPayment(amount int, addressList ...string) Payment {
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

	return Payment{
		Target: address,
		Amount: common.Amount(amount),
	}
}

func TestMakeOperation(amount int, addressList ...string) Operation {
	opb := TestMakeOperationBodyPayment(amount, addressList...)

	op := Operation{
		H: Header{
			Type: TypePayment,
		},
		B: opb,
	}

	return op
}
