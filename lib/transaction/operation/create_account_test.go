package operation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
)

func TestCreateAccountOperation(t *testing.T) {
	kp := keypair.Random()

	conf := common.NewTestConfig()
	{ // minimum Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve),
		}
		err := o.IsWellFormed(conf)
		require.NoError(t, err)
	}

	{ // insufficient Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve - 1),
		}
		err := o.IsWellFormed(conf)
		require.Equal(t, errors.InsufficientAmountNewAccount, err)
	}

	{ // sufficient Amount
		o := CreateAccount{
			Target: kp.Address(),
			Amount: common.Amount(common.BaseReserve + 1),
		}
		err := o.IsWellFormed(conf)
		require.NoError(t, err)
	}
}
