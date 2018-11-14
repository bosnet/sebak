package block

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

func TestTransactionPool(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	_, tx := transaction.TestMakeTransaction(conf.NetworkID, 1, false)

	var tp TransactionPool
	var err error
	tp, err = NewTransactionPool(tx)
	require.NoError(t, err)

	{ // save
		err = tp.Save(st)
		require.NoError(t, err)
	}

	{ // get
		rtp, err := GetTransactionPool(st, tx.GetHash())
		require.NoError(t, err)
		require.Equal(t, rtp.Hash, tx.GetHash())

		b, _ := common.EncodeJSONValue(tx)
		require.Equal(t, rtp.Message, b)
	}

	{ // delete
		err = DeleteTransactionPool(st, tx.GetHash())
		require.NoError(t, err)
	}

	{ // get; it must be failed
		_, err := GetTransactionPool(st, tx.GetHash())
		require.Error(t, err, errors.StorageRecordDoesNotExist)
	}
}
