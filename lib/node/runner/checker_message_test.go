package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

func TestMessageChecker(t *testing.T) {
	_, validTx := transaction.TestMakeTransaction(networkID, 1, false)
	var b []byte
	var err error

	if b, err = validTx.Serialize(); err != nil {
		return
	}

	validMessage := common.NetworkMessage{Type: common.TransactionMessage, Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &MessageChecker{
		DefaultChecker:  common.DefaultChecker{},
		LocalNode:       localNode,
		Consensus:       nodeRunner.Consensus(),
		Storage:         nodeRunner.Storage(),
		TransactionPool: nodeRunner.TransactionPool,
		Message:         validMessage,
		Log:             nodeRunner.Log(),
		Conf:            nodeRunner.Conf,
	}

	err = TransactionUnmarshal(checker)
	require.NoError(t, err)
	require.Equal(t, checker.Transaction, validTx)

	err = HasTransaction(checker)
	require.NoError(t, err)

	err = SaveTransactionHistory(checker)
	require.NoError(t, err)
	var found bool
	found, err = block.ExistsBlockTransactionHistory(checker.Storage, checker.Transaction.GetHash())
	require.True(t, found)

	err = PushIntoTransactionPool(checker)
	require.NoError(t, err)
	require.True(t, checker.TransactionPool.Has(validTx.GetHash()))

	// TransactionBroadcast(checker) is not suitable in unittest

	err = HasTransaction(checker)
	require.Equal(t, err, errors.NewButKnownMessage)

	err = SaveTransactionHistory(checker)
	require.Equal(t, err, errors.NewButKnownMessage)

	err = PushIntoTransactionPool(checker)
	require.NoError(t, err)

	var CheckerFuncs = []common.CheckerFunc{
		TransactionUnmarshal,
		HasTransaction,
		SaveTransactionHistory,
		PushIntoTransactionPool,
	}

	checker.DefaultChecker = common.DefaultChecker{Funcs: CheckerFuncs}

	err = common.RunChecker(checker, common.DefaultDeferFunc)
	require.Equal(t, err, errors.NewButKnownMessage)
}

func TestMessageCheckerWithInvalidHash(t *testing.T) {
	_, invalidTx := transaction.TestMakeTransaction(networkID, 1, false)
	invalidTx.H.Hash = "wrong hash"

	var b []byte
	var err error

	if b, err = invalidTx.Serialize(); err != nil {
		return
	}

	invalidMessage := common.NetworkMessage{Type: common.TransactionMessage, Data: b}
	nodeRunner, localNode := MakeNodeRunner()
	checker := &MessageChecker{
		Consensus:       nodeRunner.Consensus(),
		Storage:         nodeRunner.Storage(),
		TransactionPool: nodeRunner.TransactionPool,
		LocalNode:       localNode,
		Message:         invalidMessage,
		Log:             nodeRunner.Log(),
		Conf:            nodeRunner.Conf,
	}

	err = TransactionUnmarshal(checker)
	require.NoError(t, err)

	checker.Message.Data = []byte{}
	err = TransactionUnmarshal(checker)
	require.EqualError(t, err, "unexpected end of JSON input")
	require.NotEqual(t, checker.Transaction, invalidTx)
}

func TestMessageCheckerWithInvalidFeeSuite(t *testing.T) {
	var err error
	nodeRunner, localNode := MakeNodeRunner()
	checker := &MessageChecker{
		Consensus:       nodeRunner.Consensus(),
		Storage:         nodeRunner.Storage(),
		TransactionPool: nodeRunner.TransactionPool,
		LocalNode:       localNode,
		Log:             nodeRunner.Log(),
		Conf:            nodeRunner.Conf,
	}

	{ // valid fee
		var ops []operation.Operation
		for i := 0; i < 3; i++ {
			kp := keypair.Random()
			tba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
			tba.Save(checker.Storage)
			ops = append(ops, operation.MakeTestPaymentTo(-1, kp.Address()))
		}
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.BaseFee*3, ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.NoError(t, err)
	}

	{ // fee is over than len(Operations) * BaseFee
		var ops []operation.Operation
		for i := 0; i < 3; i++ {
			kp := keypair.Random()
			tba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
			tba.Save(checker.Storage)
			ops = append(ops, operation.MakeTestPaymentTo(-1, kp.Address()))
		}
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.BaseFee*3+1, ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.Error(t, err)
	}

	{ // fee is lower than len(Operations) * BaseFee
		var ops []operation.Operation
		for i := 0; i < 3; i++ {
			kp := keypair.Random()
			tba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
			tba.Save(checker.Storage)
			ops = append(ops, operation.MakeTestPaymentTo(-1, kp.Address()))
		}
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.BaseFee*3-1, ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.Error(t, err)
	}

	{ // zero fee transaction of normal payment operations.
		var ops []operation.Operation
		for i := 0; i < 3; i++ {
			kp := keypair.Random()
			tba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
			tba.Save(checker.Storage)
			ops = append(ops, operation.MakeTestPaymentTo(-1, kp.Address()))
		}
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.Amount(0), ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.Error(t, err)
	}

	{ // with CongressVoting, it has a fee
		var ops []operation.Operation
		opb := operation.NewCongressVoting([]byte("dummy contract"), 1, 100)
		op := operation.Operation{
			H: operation.Header{Type: operation.TypeCongressVoting},
			B: opb,
		}
		ops = append(ops, op)
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.BaseFee, ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.NoError(t, err)
	}

	{ // with CongressVoting, it has a fee
		var ops []operation.Operation
		opb := operation.NewCongressVotingResult(
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
			9, 2, 3, 4,
		)
		op := operation.Operation{
			H: operation.Header{Type: operation.TypeCongressVotingResult},
			B: opb,
		}
		ops = append(ops, op)
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.BaseFee, ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.NoError(t, err)
	}

	{ // with freezing, it is zero fee
		common.UnfreezingPeriod = 4
		var ops []operation.Operation
		tkp := keypair.Random()
		ops = append(ops, operation.MakeTestCreateFrozenAccount(100000000000, tkp.Address(), "GC4EWF2E2DPCQ5OL6EEWVCFRF5INTRB64WH4LYZW4KS7WWL6ZL5UCZ3D"))
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, false, common.Amount(0), ops)
		ba := block.NewBlockAccount(kp.Address(), common.Amount(100000000000))
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.NoError(t, err)
	}

	{ // with UnfreezeRequest, it is zero fee
		var ops []operation.Operation
		opb := operation.NewUnfreezeRequest()
		op := operation.Operation{
			H: operation.Header{Type: operation.TypeUnfreezingRequest},
			B: opb,
		}
		ops = append(ops, op)
		kp, tx := transaction.TestMakeTransactionWithFeeAndOperations(networkID, true, common.Amount(0), ops)
		ba := block.NewBlockAccount(kp.Address(), common.BaseReserve)
		ba.Linked = "GC4EWF2E2DPCQ5OL6EEWVCFRF5INTRB64WH4LYZW4KS7WWL6ZL5UCZ3D"
		ba.Save(checker.Storage)
		checker.Transaction = tx
		err = MessageValidate(checker)
		require.NoError(t, err)
	}
}
