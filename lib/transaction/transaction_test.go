package transaction

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/transaction/operation"

	"encoding/json"

	"boscoin.io/sebak/lib/error"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TransactionTestSuite struct {
	suite.Suite
	originalLimit int
}

func (suite *TransactionTestSuite) SetupTest() {
	suite.originalLimit = operation.Limit
	operation.Limit = 1000
}

func (suite *TransactionTestSuite) TeardownTest() {
	operation.Limit = suite.originalLimit
}

func (suite *TransactionTestSuite) TestLoadTransactionSuite() {
	_, tx := TestMakeTransaction(networkID, 1)

	b, err := tx.Serialize()
	require.Nil(suite.T(), err)

	var tx2 Transaction
	json.Unmarshal(b, &tx2)
	suite.T().Log(tx2)
	require.Nil(suite.T(), err)
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionSuite() {
	_, tx := TestMakeTransaction(networkID, 1)

	err := tx.IsWellFormed(networkID)
	require.Nil(suite.T(), err)
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionWithLowerFeeSuite() {
	var err error

	{ // valid fee
		kp, tx := TestMakeTransaction(networkID, 3)
		tx.Sign(kp, networkID)
		err = tx.IsWellFormed(networkID)
		require.Nil(suite.T(), err)
	}

	{ // fee is over than len(Operations) * BaseFee
		kp, tx := TestMakeTransaction(networkID, 3)
		tx.B.Fee = tx.B.Fee.MustAdd(1)
		tx.Sign(kp, networkID)
		err = tx.IsWellFormed(networkID)
		require.Nil(suite.T(), err)
	}

	{ // fee is lower than len(Operations) * BaseFee
		kp, tx := TestMakeTransaction(networkID, 3)
		tx.B.Fee = tx.B.Fee.MustSub(1)
		tx.Sign(kp, networkID)
		err = tx.IsWellFormed(networkID)
		require.Equal(suite.T(), errors.ErrorInvalidFee, err, "Transaction shouidn't pass Fee checks")
	}

	{ // zero fee
		kp, tx := TestMakeTransaction(networkID, 3)
		tx.B.Fee = common.Amount(0)
		tx.Sign(kp, networkID)
		err = tx.IsWellFormed(networkID)
		require.Equal(suite.T(), errors.ErrorInvalidFee, err, "Transaction shouidn't pass Fee checks")
	}
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionWithInvalidSourceAddressSuite() {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	tx.B.Source = "invalid-address"
	err = tx.IsWellFormed(networkID)
	require.NotNil(suite.T(), err)
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionWithTargetAddressIsSameWithSourceAddressSuite() {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	if pop, ok := tx.B.Operations[0].B.(operation.OperationBodyPayable); ok {
		tx.B.Source = pop.TargetAddress()
	} else {
		require.True(suite.T(), ok)
	}
	err = tx.IsWellFormed(networkID)
	require.NotNil(suite.T(), err, "Transaction to self should be rejected")
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionWithInvalidSignatureSuite() {
	var err error

	_, tx := TestMakeTransaction(networkID, 1)
	err = tx.IsWellFormed(networkID)
	require.Nil(suite.T(), err)

	newSignature, _ := keypair.Master("find me").Sign(append(networkID, []byte(tx.B.MakeHashString())...))
	tx.H.Signature = base58.Encode(newSignature)

	err = tx.IsWellFormed(networkID)
	require.NotNil(suite.T(), err)
}

func (suite *TransactionTestSuite) TestIsWellFormedTransactionMaxOperationsInTransactionSuite() {
	var err error

	{ // over operation.Limit
		_, tx := TestMakeTransaction(networkID, operation.Limit+1)
		err = tx.IsWellFormed(networkID)
		require.Equal(suite.T(), errors.ErrorTransactionHasOverMaxOperations, err)
	}

	{ // operation.Limit
		_, tx := TestMakeTransaction(networkID, operation.Limit)
		err = tx.IsWellFormed(networkID)
		require.Nil(suite.T(), err)
	}
}

func TestTransaction(t *testing.T) {
	suite.Run(t, new(TransactionTestSuite))
}
