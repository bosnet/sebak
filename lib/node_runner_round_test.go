package sebak

import (
	"sync"
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"github.com/stellar/go/keypair"
)

func createNodeRunnerRounds(n int) []*NodeRunnerRound {
	var ns []*sebaknetwork.MemoryNetwork
	var nodes []*sebaknode.LocalNode
	for i := 0; i < n; i++ {
		s, v := createNetMemoryNetwork()
		ns = append(ns, s)
		nodes = append(nodes, v)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			nodes[i].AddValidators(nodes[j].ConvertToValidator())
		}
	}

	var nodeRunners []*NodeRunnerRound
	for i := 0; i < n; i++ {
		v := nodes[i]
		p, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
		p.SetValidators(len(v.GetValidators()) + 1)
		is, _ := NewISAACRound(networkID, v, p)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
		nr, err := NewNodeRunnerRound(string(networkID), v, p, ns[i], is, st)
		if err != nil {
			panic(err)
		}
		nodeRunners = append(nodeRunners, nr)
	}

	return nodeRunners
}

func createNodeRunnerRoundsWithReady(n int) []*NodeRunnerRound {
	nodeRunners := createNodeRunnerRounds(n)

	for _, nr := range nodeRunners {
		go nr.Start()
	}

	T := time.NewTicker(100 * time.Millisecond)
	stopTimer := make(chan bool)

	go func() {
		time.Sleep(5 * time.Second)
		stopTimer <- true
	}()

	go func() {
		for _ = range T.C {
			var notyet bool
			for _, nr := range nodeRunners {
				if nr.ConnectionManager().CountConnected() != n-1 {
					notyet = true
					break
				}
			}
			if notyet {
				continue
			}
			stopTimer <- true
		}
	}()
	select {
	case <-stopTimer:
		T.Stop()
	}

	return nodeRunners
}

func TestCreateNodeRunnerRounds(t *testing.T) {
	nodeRunners := createNodeRunnerRoundsWithReady(3)

	if len(nodeRunners) != 3 {
		t.Error("failed to create `NodeRunnerRound`s")
	}
}

func TestNodeRunnerRoundCreateAccount(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnerRoundsWithReady(numberOfNodes)
	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	kp, _ := keypair.Random()
	kpNewAccount, _ := keypair.Random()

	// create new account in all nodes
	var account *BlockAccount
	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	for _, nr := range nodeRunners {
		address := kp.Address()
		balance := BaseFee.MustAdd(1)

		account = NewBlockAccount(address, balance, checkpoint)
		account.Save(nr.Storage())
	}

	var wg sync.WaitGroup

	wg.Add(numberOfNodes)

	var dones []VotingStateStaging
	var finished []string
	var deferFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}

		checker := c.(*NodeRunnerHandleBallotChecker)
		if _, found := sebakcommon.InStringArray(finished, checker.LocalNode.Alias()); found {
			return
		}
		finished = append(finished, checker.LocalNode.Alias())
		dones = append(dones, checker.VotingStateStaging)
		wg.Done()
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotFuncs(deferFunc, nil)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := Amount(1)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()
}
