package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
)

func TestOnlyValidTransactionInTransactionPool(t *testing.T) {
	nodeRunners, rootKP := createTestNodeRunnersHTTP2NetworkWithReady(3)
	nodeRunner := nodeRunners[0]

	rootAccount, _ := block.GetBlockAccount(nodeRunner.Storage(), rootKP.Address())

	TestMakeBlockAccount := func(balance sebakcommon.Amount) (account *block.BlockAccount, kp *keypair.Full) {
		kp, _ = keypair.Random()
		account = block.NewBlockAccount(kp.Address(), balance, TestGenerateNewCheckpoint())

		return
	}

	runChecker := func(tx Transaction, expectedError error) {
		messageData, _ := tx.Serialize()

		checker := &NodeRunnerHandleMessageChecker{
			DefaultChecker: sebakcommon.DefaultChecker{DefaultHandleMessageFromClientCheckerFuncs},
			NodeRunner:     nodeRunner,
			LocalNode:      nodeRunner.Node(),
			NetworkID:      networkID,
			Message:        sebaknetwork.Message{Type: "message", Data: messageData},
		}

		if err := sebakcommon.RunChecker(checker, nil); err != nil {
			if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
				if expectedError != nil && err != expectedError {
					t.Error("error must be", expectedError, "but found", err)
					return
				}
				log.Error("failed to handle message", "error", err)
			}
		}
	}

	{ // valid transaction
		targetAccount, targetKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000) /* 100,00000 BOS */)
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.Checkpoint = rootAccount.Checkpoint
		tx.Sign(rootKP, networkID)

		runChecker(tx, nil)

		if !nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()) {
			t.Error("valid transaction must be in `TransactionPool`")
			return
		}
	}

	{ // invalid transaction: same source already in TransactionPool
		targetAccount, targetKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, rootKP, targetKP)
		tx.B.Checkpoint = rootAccount.Checkpoint
		tx.Sign(rootKP, networkID)

		runChecker(tx, sebakerror.ErrorTransactionSameSource)

		if nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()) {
			t.Error("invalid transaction must be in `TransactionPool`: same source already in `TransactionPool`")
			return
		}
	}

	{ // invalid transaction: source account does not exists
		_, sourceKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000))
		targetAccount, targetKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000))
		targetAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)

		runChecker(tx, sebakerror.ErrorBlockAccountDoesNotExists)

		if nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()) {
			t.Error("invalid transaction must be in `TransactionPool`: source account does not exists")
			return
		}
	}

	{ // invalid transaction: target account does not exists
		sourceAccount, sourceKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000))
		_, targetKP := TestMakeBlockAccount(sebakcommon.Amount(10000000000000))
		sourceAccount.Save(nodeRunner.Storage())

		tx := TestMakeTransactionWithKeypair(networkID, 1, sourceKP, targetKP)
		tx.B.Checkpoint = sourceAccount.Checkpoint
		tx.Sign(sourceKP, networkID)

		runChecker(tx, sebakerror.ErrorBlockAccountDoesNotExists)

		if nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()) {
			t.Error("invalid transaction must be in `TransactionPool`: target account does not exists")
			return
		}
	}
}
