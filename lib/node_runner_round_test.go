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

	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	kp, _ := keypair.Random()
	address := kp.Address()
	balance := BaseFee.MustAdd(1)
	account := NewBlockAccount(address, balance, checkpoint)
	var nodeRunners []*NodeRunnerRound
	for i := 0; i < n; i++ {
		v := nodes[i]
		p, _ := NewDefaultVotingThresholdPolicy(100, 66, 66)
		p.SetValidators(len(v.GetValidators()) + 1)
		is, _ := NewISAACRound(networkID, v, p)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

		account.Save(st)
		MakeGenesisBlock(st, *account)

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
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnerRoundsWithReady(numberOfNodes)

	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	if len(nodeRunners) != 3 {
		t.Error("failed to create `NodeRunnerRound`s")
	}
}

func TestNodeRunnerRoundCreateAccount(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createNodeRunnerRoundsWithReady(numberOfNodes)
	kpNewAccount, _ := keypair.Random()

	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	var wg sync.WaitGroup
	wg.Add(numberOfNodes)

	results := map[string]map[string]Transaction{}

	var finishedFunc sebakcommon.CheckerDeferFunc = func(n int, c sebakcommon.Checker, err error) {
		if err == nil {
			return
		}

		checker := c.(*NodeRunnerRoundHandleBallotChecker)

		results[checker.LocalNode.Address()] = checker.NodeRunner.Consensus().TransactionPool
		wg.Done()

		return
	}

	for _, nr := range nodeRunners {
		nr.SetHandleBallotFuncs(nil, finishedFunc)
	}

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := Amount(1)
	kp, _ := keypair.Random()
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)

	checkpoint := sebakcommon.MakeGenesisCheckpoint(networkID)
	tx.B.Checkpoint = checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	wg.Wait()

	for _, nr := range nodeRunners {
		nr.Stop()
	}

	for _, nr := range nodeRunners {
		txpool, found := results[nr.Node().Address()]
		if !found {
			t.Error("failed to broadcast message; `TransactionPool` is empty")
			return
		}
		if len(txpool) != 1 {
			t.Error("failed to broadcast message; `TransactionPool` is filled with other messages")
			return
		}

		if _, found := txpool[tx.GetHash()]; !found {
			t.Error("failed to broadcast message; tx not found in `TransactionPool`")
			return
		}
	}
}
