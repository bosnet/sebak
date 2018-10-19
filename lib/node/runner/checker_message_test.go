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
		LocalNode:      localNode,
		Consensus:      nodeRunner.Consensus(),
		Storage:        nodeRunner.Storage(),
		TransactionPool: nodeRunner.TransactionPool,
		NetworkID:      networkID,
		Message:        validMessage,
		Log:            nodeRunner.Log(),
		Conf:           nodeRunner.Conf,
	}

	err = TransactionUnmarshal(checker)
	require.Nil(t, err)
	require.Equal(t, checker.Transaction, validTx)

	err = HasTransaction(checker)
	require.Nil(t, err)

	err = SaveTransactionHistory(checker)
	require.Nil(t, err)
	var found bool
	found, err = block.ExistsBlockTransactionHistory(checker.Storage, checker.Transaction.GetHash())
	require.True(t, found)

	err = PushIntoTransactionPool(checker)
	require.Nil(t, err)
	require.True(t, checker.TransactionPool.Has(validTx.GetHash()))

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

func TestMessageCheckerWithInvalidHash(t *testing.T) {
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
		Consensus: nodeRunner.Consensus(),
		Storage:   nodeRunner.Storage(),
		TransactionPool: nodeRunner.TransactionPool,
		LocalNode: localNode,
		NetworkID: networkID,
		Message:   invalidMessage,
		Log:       nodeRunner.Log(),
		Conf:      nodeRunner.Conf,
	}

	err = TransactionUnmarshal(checker)
	require.Nil(t, err)

	checker.Message.Data = []byte{}
	err = TransactionUnmarshal(checker)
	require.EqualError(t, err, "unexpected end of JSON input")
	require.NotEqual(t, checker.Transaction, invalidTx)
}
