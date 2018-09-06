package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
)

func TestOnlyValidTransactionInTransactionPool(t *testing.T) {
	nodeRunners, rootKP := createTestNodeRunnersHTTP2NetworkWithReady(3)
	nodeRunner := nodeRunners[0]

	rootAccount, _ := block.GetBlockAccount(nodeRunner.Storage(), rootKP.Address())

	TestMakeBlockAccount := func(balance common.Amount) (account *block.BlockAccount, kp *keypair.Full) {
		kp, _ = keypair.Random()
		account = block.NewBlockAccount(kp.Address(), balance, TestGenerateNewCheckpoint())

		return
	}

	runChecker := func(tx Transaction, expectedError error) {
		messageData, _ := tx.Serialize()

		checker := &MessageChecker{
			DefaultChecker: common.DefaultChecker{Funcs: DefaultHandleTransactionCheckerFuncs},
			NodeRunner:     nodeRunner,
			LocalNode:      nodeRunner.Node(),
			NetworkID:      networkID,
			Message:        network.Message{Type: "message", Data: messageData},
		}

		if err := common.RunChecker(checker, nil); err != nil {
			if _, ok := err.(common.CheckerErrorStop); !ok && expectedError != nil {
				require.Error(t, err, expectedError)
			}
		}
	}

	{ // valid transaction
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000) /* 100,00000 BOS */)
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.Checkpoint = rootAccount.Checkpoint
		tx.Sign(rootKP, networkID)

		runChecker(tx, nil)

		require.True(t, nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()), "valid transaction must be in `TransactionPool`")
	}

	{ // invalid transaction: same source already in TransactionPool
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.Checkpoint = rootAccount.Checkpoint
		tx.Sign(rootKP, networkID)

		runChecker(tx, errors.ErrorTransactionSameSource)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `TransactionPool`: same source already in `TransactionPool`",
		)
	}

	{ // invalid transaction: source account does not exists
		_, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)

		runChecker(tx, errors.ErrorBlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must not be in `TransactionPool`: source account does not exists",
		)
	}

	{ // invalid transaction: target account does not exists
		sourceAccount, sourceKP := TestMakeBlockAccount(common.Amount(10000000000000))
		_, targetKP := TestMakeBlockAccount(common.Amount(10000000000000))
		sourceAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)
		tx.B.Checkpoint = sourceAccount.Checkpoint
		tx.Sign(sourceKP, networkID)

		runChecker(tx, errors.ErrorBlockAccountDoesNotExists)

		require.False(
			t,
			nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()),
			"invalid transaction must be in `TransactionPool`: target account does not exists",
		)
	}
}
