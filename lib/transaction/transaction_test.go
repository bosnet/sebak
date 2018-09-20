package transaction

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"encoding/json"
)

func TestLoadTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)

	b, err := tx.Serialize()
	require.Nil(t, err)


	var tx2 Transaction
	json.Unmarshal(b, &tx2)
	t.Log(tx2)
	require.Nil(t, err)
}


func TestIsWellFormedTransaction(t *testing.T) {
	_, tx := TestMakeTransaction(networkID, 1)

	err := tx.IsWellFormed(networkID)
	require.Nil(t, err)
}

func TestIsWellFormedTransactionWithLowerFee(t *testing.T) {
	var err error

	kp, tx := TestMakeTransaction(networkID, 1)
	tx.B.Fee = common.BaseFee
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	err = tx.IsWellFormed(networkID)
	require.Nil(t, err)

	tx.B.Fee = common.BaseFee.MustAdd(1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	err = tx.IsWellFormed(networkID)
	require.Nil(t, err)

	tx.B.Fee = common.BaseFee.MustSub(1)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	err = tx.IsWellFormed(networkID)
	require.NotNil(t, err, "Transaction shouidn't pass Fee checks")

	tx.B.Fee = common.Amount(0)
	tx.H.Hash = tx.B.MakeHashString()
	tx.Sign(kp, networkID)
	err = tx.IsWellFormed(networkID)
	require.NotNil(t, err, "Transaction shouidn't pass Fee checks")
}

func TestIsWellFormedTransactionWithInvalidSourceAddress(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	tx.B.Source = "invalid-address"
	err = tx.IsWellFormed(networkID)
	require.NotNil(t, err)
}

func TestIsWellFormedTransactionWithTargetAddressIsSameWithSourceAddress(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	tx.B.Source = tx.B.Operations[0].B.TargetAddress()
	err = tx.IsWellFormed(networkID)
	require.NotNil(t, err, "Transaction to self should be rejected")
}

func TestIsWellFormedTransactionWithInvalidSignature(t *testing.T) {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	err = tx.IsWellFormed(networkID)
	require.Nil(t, err)

	newSignature, _ := keypair.Master("find me").Sign(append(networkID, []byte(tx.B.MakeHashString())...))
	tx.H.Signature = base58.Encode(newSignature)

	err = tx.IsWellFormed(networkID)
	require.NotNil(t, err)
}

func TestIsWellFormedTransactionMaxOperationsInTransaction(t *testing.T) {
	var err error

	{ // over common.MaxOperationsInTransaction
		_, tx := TestMakeTransaction(networkID, common.MaxOperationsInTransaction+1)
		err = tx.IsWellFormed(networkID)
		require.Equal(t, errors.ErrorTransactionHasOverMaxOperations, err)
	}

	{ // common.MaxOperationsInTransaction
		_, tx := TestMakeTransaction(networkID, common.MaxOperationsInTransaction)
		err = tx.IsWellFormed(networkID)
		require.Nil(t, err)
	}
}
