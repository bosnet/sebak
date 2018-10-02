package operation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

func TestCreateAccountOperation(t *testing.T) {
	{ // minimum Amount
		o := OperationBodyCreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve),
		}
		err := o.IsWellFormed(networkID)
		require.Nil(t, err)
	}

	{ // insufficient Amount
		o := OperationBodyCreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve - 1),
		}
		err := o.IsWellFormed(networkID)
		require.Equal(t, errors.ErrorInsufficientAmountNewAccount, err)
	}

	{ // sufficient Amount
		o := OperationBodyCreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve + 1),
		}
		err := o.IsWellFormed(networkID)
		require.Nil(t, err)
	}
}
