package operation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

func TestCreateAccountOperation(t *testing.T) {

	conf := common.NewConfig()
	{ // minimum Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve),
		}
		err := o.IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}

	{ // insufficient Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve - 1),
		}
		err := o.IsWellFormed(networkID, conf)
		require.Equal(t, errors.InsufficientAmountNewAccount, err)
	}

	{ // sufficient Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve + 1),
		}
		err := o.IsWellFormed(networkID, conf)
		require.NoError(t, err)
	}
}
