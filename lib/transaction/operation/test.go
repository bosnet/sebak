package operation

import (
	"math/rand"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
)

func MakeTestPayment(amount int) Operation {
	return MakeTestPaymentTo(amount, keypair.Random().Address())
}

func MakeTestPaymentTo(amount int, address string) Operation {
	for amount < 0 {
		amount = rand.Intn(5000)
	}

	return Operation{
		H: Header{
			Type: TypePayment,
		},
		B: Payment{
			Target: address,
			Amount: common.Amount(amount),
		},
	}
}

func MakeTestUnfreezeRequest() Operation {
	return Operation{
		H: Header{
			Type: TypeUnfreezingRequest,
		},
		B: UnfreezeRequest{},
	}
}

func MakeTestCreateAccount(amount int) Operation {
	return MakeTestCreateFrozenAccount(amount, keypair.Random().Address(), "")
}

func MakeTestCreateFrozenAccount(amount int, address string, linked string) Operation {
	for amount < 0 {
		amount = rand.Intn(5000)
	}
	return Operation{
		H: Header{
			Type: TypeCreateAccount,
		},
		B: CreateAccount{
			Target: address,
			Amount: common.Amount(amount),
			Linked: linked,
		},
	}
}
