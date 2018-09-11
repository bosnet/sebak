package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
)

func TestMessageChecker(t *testing.T) {
	_, validTx := transaction.TestMakeTransaction(networkID, 1)
	var b []byte
	var err error

	if b, err = validTx.Serialize(); err != nil {
		return
	}

	validMessage := common.NetworkMessage{Type: common.TransactionMessage, Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &MessageChecker{
		DefaultChecker: common.DefaultChecker{},
		NodeRunner:     nodeRunner,
		LocalNode:      localNode,
		NetworkID:      networkID,
		Message:        validMessage,
	}

	err = TransactionUnmarshal(checker)
	require.Nil(t, err)
	require.Equal(t, checker.Transaction, validTx)

	err = HasTransaction(checker)
	require.Nil(t, err)

	err = SaveTransactionHistory(checker)
	require.Nil(t, err)
	var found bool
	found, err = block.ExistsBlockTransactionHistory(checker.NodeRunner.Storage(), checker.Transaction.GetHash())
	require.True(t, found)

	err = PushIntoTransactionPool(checker)
	require.Nil(t, err)
	require.True(t, checker.NodeRunner.Consensus().TransactionPool.Has(validTx.GetHash()))

	// TransactionBroadcast(checker) is not suitable in unittest

	err = HasTransaction(checker)
	require.Equal(t, err, errors.ErrorNewButKnownMessage)

	err = SaveTransactionHistory(checker)
	require.Equal(t, err, errors.ErrorNewButKnownMessage)

	err = PushIntoTransactionPool(checker)
	require.Nil(t, err)

	var CheckerFuncs = []common.CheckerFunc{
		TransactionUnmarshal,
		HasTransaction,
		SaveTransactionHistory,
		PushIntoTransactionPool,
	}

	checker.DefaultChecker = common.DefaultChecker{Funcs: CheckerFuncs}

	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Equal(t, err, errors.ErrorNewButKnownMessage)
}

func TestMessageCheckerWithInvalidMessage(t *testing.T) {
	_, invalidTx := transaction.TestMakeTransaction(networkID, 1)
	invalidTx.H.Hash = "wrong hash"

	var b []byte
	var err error

	if b, err = invalidTx.Serialize(); err != nil {
		return
	}

	invalidMessage := common.NetworkMessage{Type: common.TransactionMessage, Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &MessageChecker{
		NodeRunner: nodeRunner,
		LocalNode:  localNode,
		NetworkID:  networkID,
		Message:    invalidMessage,
	}

	err = TransactionUnmarshal(checker)
	require.EqualError(t, err, errors.ErrorSignatureVerificationFailed.Message)

	checker.Message.Data = []byte{}
	err = TransactionUnmarshal(checker)
	require.EqualError(t, err, "unexpected end of JSON input")
	require.NotEqual(t, checker.Transaction, invalidTx)

}
