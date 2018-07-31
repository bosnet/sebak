package sebak

import (
	"sync"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
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

// TestNodeRunnerConsensusSameSourceWillBeIgnored checks, the transaction which
// has same source will be ignored if the transaction has same source and it is
// in 'SIGN' state.
func TestNodeRunnerConsensusSameSourceWillBeIgnored(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	var wg sync.WaitGroup

	nr0 := nodeRunners[0]
	firstTx := makeTransaction(nr0.Node().Keypair())
	secondTx := makeTransaction(nr0.Node().Keypair())
	var mutex = &sync.Mutex{}
	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,

		// stop consensus when reached 'SIGN'
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			checker := c.(*NodeRunnerHandleBallotChecker)

			if checker.Ballot.MessageHash() != firstTx.GetHash() {
				return
			}

			if checker.VotingStateStaging.State == sebakcommon.BallotStateSIGN {
				mutex.Lock()
				defer mutex.Unlock()

				err = sebakcommon.CheckerErrorStop{"stop consensus, because it is in SIGN state"}
				wg.Done()
				return
			}

			return
		},
		CheckNodeRunnerHandleBallotHistory,
		CheckNodeRunnerHandleBallotStore,
		CheckNodeRunnerHandleBallotIsBroadcastable,
		CheckNodeRunnerHandleBallotVotingHole,
		CheckNodeRunnerHandleBallotBroadcast,
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(nil, handleBallotCheckerFuncs...)
	}

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	wg.Add(numberOfNodes)
	client.SendMessage(firstTx)
	wg.Wait()

	isaac := nr0.Consensus().(*ISAAC)
	if !isaac.HasMessage(firstTx) {
		t.Error("transaction not found")
		return
	}

	if _, ok := isaac.Boxes.Results[firstTx.GetHash()]; !ok {
		t.Error("VotingResult not found")
		return
	}

	if !isaac.Boxes.VotingBox.HasMessage(firstTx) {
		t.Error("ballot not in VotingBox")
		return
	}

	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			mutex.Lock()
			defer mutex.Unlock()
			wg.Done()
			return
		}
	}

	for _, nr := range nodeRunners {
		nr.SetHandleMessageFromClientCheckerFuncs(deferFunc)
	}

	wg = sync.WaitGroup{}
	wg.Add(1)
	client.SendMessage(secondTx)
	wg.Wait()

	if isaac.HasMessage(secondTx) {
		t.Error("second transaction was added as VotingResult")
		return
	}
}

// TestNodeRunnerConsensusSameSourceWillNotIgnored checks, the transaction which
// has same source will be ignored if the transaction has same source and it is
// not in 'SIGN' state.
func TestNodeRunnerConsensusSameSourceWillNotIgnored(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnersWithReady(numberOfNodes)

	var wg sync.WaitGroup

	nr0 := nodeRunners[0]
	firstTx := makeTransaction(nr0.Node().Keypair())
	secondTx := makeTransaction(nr0.Node().Keypair())

	var handleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,

		// stop consensus when reached 'INIT'
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			err = sebakcommon.CheckerErrorStop{"stop consensus, because it is in INIT state"}
			defer wg.Done()
			return
		},
		CheckNodeRunnerHandleBallotHistory,
		CheckNodeRunnerHandleBallotStore,
		CheckNodeRunnerHandleBallotIsBroadcastable,
		// instead of `CheckNodeRunnerHandleBallotVotingHole`
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			checker := c.(*NodeRunnerHandleBallotChecker)

			checker.VotingHole = VotingYES

			return
		},
		CheckNodeRunnerHandleBallotBroadcast,
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(nil, handleBallotCheckerFuncs...)
	}

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	wg.Add(2)
	client.SendMessage(firstTx)
	wg.Wait()

	isaac := nr0.Consensus().(*ISAAC)
	if !isaac.HasMessage(firstTx) {
		t.Error("transaction not found")
		return
	}

	if _, ok := isaac.Boxes.Results[firstTx.GetHash()]; !ok {
		t.Error("VotingResult not found")
		return
	}

	if !isaac.Boxes.WaitingBox.HasMessage(firstTx) {
		t.Error("ballot not in WaitingBox")
		return
	}

	var finished []string
	var dones []VotingStateStaging
	var mutex = &sync.Mutex{}
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if checker.VotingStateStaging.IsEmpty() {
			return
		}

		if !checker.VotingStateStaging.IsClosed() {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()

		if _, found := sebakcommon.InStringArray(finished, checker.LocalNode.Alias()); found {
			return
		}
		finished = append(finished, checker.LocalNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	var secondHandleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
		CheckNodeRunnerHandleBallotIsWellformed,
		CheckNodeRunnerHandleBallotCheckIsNew,
		CheckNodeRunnerHandleBallotReceiveBallot,
		CheckNodeRunnerHandleBallotHistory,
		// skip `CheckNodeRunnerHandleBallotStore`
		CheckNodeRunnerHandleBallotIsBroadcastable,
		// instead of `CheckNodeRunnerHandleBallotVotingHole`
		func(c sebakcommon.Checker, args ...interface{}) (err error) {
			checker := c.(*NodeRunnerHandleBallotChecker)
			checker.VotingHole = VotingYES

			return
		},
		CheckNodeRunnerHandleBallotBroadcast,
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotCheckerFuncs(deferFunc, secondHandleBallotCheckerFuncs...)
	}

	wg = sync.WaitGroup{}
	wg.Add(numberOfNodes)
	client.SendMessage(secondTx)
	wg.Wait()

	if len(dones) != numberOfNodes {
		t.Error("failed to get consensus")
		return
	}

	for _, vs := range dones {
		if !vs.IsClosed() {
			t.Error("failed to close consensus")
			return
		}
	}
}
