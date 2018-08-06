package sebak

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"github.com/stellar/go/keypair"
)

var (
	kp      *keypair.Full
	account *BlockAccount
)

func init() {
	kp, _ = keypair.Random()
}

func createTestNodeRunner(n int) []*NodeRunner {
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
	address := kp.Address()
	balance := BaseFee.MustAdd(1)
	account = NewBlockAccount(address, balance, checkpoint)
	var nodeRunners []*NodeRunner
	for i := 0; i < n; i++ {
		v := nodes[i]
		p, _ := NewDefaultVotingThresholdPolicy(66)
		p.SetValidators(len(v.GetValidators()) + 1)
		is, _ := NewISAAC(networkID, v, p)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

		account.Save(st)
		MakeGenesisBlock(st, *account)

		nr, err := NewNodeRunner(string(networkID), v, p, ns[i], is, st)
		if err != nil {
			panic(err)
		}
		nodeRunners = append(nodeRunners, nr)
	}

	return nodeRunners
}

func createTestNodeRunnerWithReady(n int) []*NodeRunner {
	nodeRunners := createTestNodeRunner(n)

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

func TestCreateNodeRunner(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createTestNodeRunnerWithReady(numberOfNodes)

	for _, nr := range nodeRunners {
		defer nr.Stop()
	}

	if len(nodeRunners) != 3 {
		t.Error("failed to create `NodeRunner`s")
	}
}

func TestNodeRunnerCreateAccount(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createTestNodeRunnerWithReady(numberOfNodes)

	kpNewAccount, _ := keypair.Random()

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := Amount(1)
	tx := makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp, networkID)

	client.SendMessage(tx)

	time.Sleep(time.Second)

	for _, nr := range nodeRunners {
		nr.Stop()
	}

	for i, nr := range nodeRunners {
		if !nr.Consensus().TransactionPool.Has(tx.GetHash()) {
			t.Error("failed to broadcast message", "node", nr.Node(), "index", i)
		}
	}
}
