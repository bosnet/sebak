package sebak

import (
	"sync"
	"testing"

	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/network"
)

// TestNodeRunnerConsensusStoreInHistoryIncomingTxMessage checks, the incoming tx message will be
// saved in 'BlockTransactionHistory'.
func TestNodeRunnerConsensusStoreInHistoryIncomingTxMessage(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	var wg sync.WaitGroup
	wg.Add(1)

	var handleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleMessageTransactionUnmarshal,
		CheckNodeRunnerHandleMessageHistory,
		func(c sebakcommon.Checker, args ...interface{}) error {
			defer wg.Done()

			return nil
		},
	}

	for _, nr := range nodeRunners {
		nr.SetHandleMessageFromClientCheckerFuncs(nil, handleMessageFromClientCheckerFuncs...)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())
	tx := makeTransaction(nr0.Node().Keypair())
	client.SendMessage(tx)

	wg.Wait()

	if nr0.Consensus().HasMessageByHash(tx.GetHash()) {
		t.Error("failed to close consensus; message still in consensus")
		return
	}

	{
		history, err := GetBlockTransactionHistory(nr0.Storage(), tx.GetHash())
		if err != nil {
			t.Error(err)
			return
		}
		if history.Hash != tx.GetHash() {
			t.Error("saved invalid hash")
			return
		}
	}
}

// TestNodeRunnerConsensusStoreInHistoryNewBallot checks, the incoming new
// ballot will be saved in 'BlockTransactionHistory'.
func TestNodeRunnerConsensusStoreInHistoryNewBallot(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	var wg sync.WaitGroup
	wg.Add(2)

	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,
		CheckNodeRunnerHandleBallotHistory,
		func(c sebakcommon.Checker, args ...interface{}) error {
			checker := c.(*NodeRunnerHandleBallotChecker)
			if !checker.IsNew {
				return nil
			}
			wg.Done()
			return nil
		},
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(nil, handleBallotCheckerFuncs...)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	tx := makeTransaction(nr0.Node().Keypair())
	client.SendMessage(tx)

	wg.Wait()

	for _, nr := range nodeRunners {
		if nr.Node() == nr0.Node() {
			continue
		}

		history, err := GetBlockTransactionHistory(nr.Storage(), tx.GetHash())
		if err != nil {
			t.Error(err)
			return
		}
		if history.Hash != tx.GetHash() {
			t.Error("saved invalid hash")
			return
		}
	}
}
