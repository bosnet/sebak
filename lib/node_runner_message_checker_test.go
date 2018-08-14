package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

func MakeNodeRunner() (*NodeRunner, *sebaknode.LocalNode) {
	kp, _ = keypair.Random()

	nodeEndpoint := &sebakcommon.Endpoint{Scheme: "https", Host: "https://locahost:5000"}
	localNode, _ := sebaknode.NewLocalNode(kp, nodeEndpoint, "")

	vth, _ := NewDefaultVotingThresholdPolicy(66, 66)
	is, _ := NewISAAC(networkID, localNode, vth)
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	network, _ := createNetMemoryNetwork()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, vth, network, is, st)
	return nodeRunner, localNode
}

func TestMessageChecker(t *testing.T) {
	var CheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageHasTransactionAlready,
		CheckNodeRunnerHandleMessageHistory,
		CheckNodeRunnerHandleMessagePushIntoTransactionPool,
		CheckNodeRunnerHandleMessageTransactionBroadcast,
	}

	_, validTx := TestMakeTransaction(networkID, 1)
	var b []byte
	var err error

	if b, err = validTx.Serialize(); err != nil {
		return
	}

	validMessage := sebaknetwork.Message{Type: "message", Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &NodeRunnerHandleMessageChecker{
		DefaultChecker: sebakcommon.DefaultChecker{CheckerFuncs},
		NodeRunner:     nodeRunner,
		LocalNode:      localNode,
		NetworkID:      networkID,
		Message:        validMessage,
	}

	err = CheckNodeRunnerHandleMessageTransactionUnmarshal(checker)
	assert.Nil(t, err)
	assert.Equal(t, checker.Transaction, validTx)

	err = CheckNodeRunnerHandleMessageHasTransactionAlready(checker)
	assert.Nil(t, err)

	err = CheckNodeRunnerHandleMessageHistory(checker)
	assert.Nil(t, err)
	var found bool
	found, err = ExistsBlockTransactionHistory(checker.NodeRunner.Storage(), checker.Transaction.GetHash())
	assert.True(t, found)

	err = CheckNodeRunnerHandleMessagePushIntoTransactionPool(checker)
	assert.Nil(t, err)
	assert.True(t, checker.NodeRunner.Consensus().TransactionPool.Has(validTx.GetHash()))

	// CheckNodeRunnerHandleMessageTransactionBroadcast(checker) is not suitable in unittest

	err = CheckNodeRunnerHandleMessageHasTransactionAlready(checker)
	assert.Equal(t, err, sebakerror.ErrorNewButKnownMessage)

	err = CheckNodeRunnerHandleMessageHistory(checker)
	assert.Equal(t, err, sebakerror.ErrorNewButKnownMessage)

	err = CheckNodeRunnerHandleMessagePushIntoTransactionPool(checker)
	assert.Nil(t, err)

	err = sebakcommon.RunChecker(checker, sebakcommon.DefaultDeferFunc)
	assert.Equal(t, err, sebakerror.ErrorNewButKnownMessage)
}

func TestMessageCheckerWithInvalidMessage(t *testing.T) {
	_, invalidTx := TestMakeTransaction(networkID, 1)
	invalidTx.H.Hash = "wrong hash"

	var b []byte
	var err error

	if b, err = invalidTx.Serialize(); err != nil {
		return
	}

	invalidMessage := sebaknetwork.Message{Type: "message", Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &NodeRunnerHandleMessageChecker{
		NodeRunner: nodeRunner,
		LocalNode:  localNode,
		NetworkID:  networkID,
		Message:    invalidMessage,
	}

	err = CheckNodeRunnerHandleMessageTransactionUnmarshal(checker)
	assert.EqualError(t, err, sebakerror.ErrorSignatureVerificationFailed.Message)

	checker.Message.Data = []byte{}
	err = CheckNodeRunnerHandleMessageTransactionUnmarshal(checker)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.NotEqual(t, checker.Transaction, invalidTx)

}
