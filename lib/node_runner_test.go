package sebak

import (
	"reflect"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"github.com/stellar/go/keypair"
)

var (
	kp      *keypair.Full
	account *block.BlockAccount
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
	account = block.NewBlockAccount(address, balance, checkpoint)
	var nodeRunners []*NodeRunner
	for i := 0; i < n; i++ {
		v := nodes[i]
		p, _ := NewDefaultVotingThresholdPolicy(66, 66)
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

	initialBalance := sebakcommon.Amount(1)
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

func TestNodeRunnersHaveSameProposers(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createTestNodeRunnerWithReady(numberOfNodes)

	nr0 := nodeRunners[0]
	nr1 := nodeRunners[1]
	nr2 := nodeRunners[2]

	var maximumBlockHeight uint64 = 3
	var maximumRoundNumber uint64 = 3

	proposers0 := make([]string, maximumBlockHeight*maximumRoundNumber)
	proposers1 := make([]string, maximumBlockHeight*maximumRoundNumber)
	proposers2 := make([]string, maximumBlockHeight*maximumRoundNumber)

	for i := uint64(0); i < maximumBlockHeight; i++ {
		for j := uint64(0); j < maximumRoundNumber; j++ {
			proposers0[i*maximumRoundNumber+j] = nr0.CalculateProposer(i, j)
			proposers1[i*maximumRoundNumber+j] = nr1.CalculateProposer(i, j)
			proposers2[i*maximumRoundNumber+j] = nr2.CalculateProposer(i, j)
		}
	}

	if !reflect.DeepEqual(proposers0, proposers1) {
		t.Error("failed to have same proposers. nr0, nr1.")
	}
	if !reflect.DeepEqual(proposers0, proposers2) {
		t.Error("failed to have same proposers. nr0, nr2.")
	}
	if !reflect.DeepEqual(proposers1, proposers2) {
		t.Error("failed to have same proposers. nr1, nr2.")
	}

	for _, nr := range nodeRunners {
		nr.Stop()
	}
}

func TestNodeRunnerHasEvenProposers(t *testing.T) {
	defer sebaknetwork.CleanUpMemoryNetwork()

	numberOfNodes := 3
	nodeRunners := createTestNodeRunnerWithReady(numberOfNodes)

	nr0 := nodeRunners[0]
	nr1 := nodeRunners[1]
	nr2 := nodeRunners[2]

	var maximumBlockHeight uint64 = 3
	var maximumRoundNumber uint64 = 10

	proposers0 := make([]string, maximumBlockHeight*maximumRoundNumber)

	for i := uint64(0); i < maximumBlockHeight; i++ {
		for j := uint64(0); j < maximumRoundNumber; j++ {
			proposers0[i*maximumRoundNumber+j] = nr0.CalculateProposer(i, j)
		}
	}

	numN0inProposers := 0
	numN1inProposers := 0
	numN2inProposers := 0

	for _, p := range proposers0 {
		if p == nr0.localNode.Address() {
			numN0inProposers++
		} else if p == nr1.localNode.Address() {
			numN1inProposers++
		} else if p == nr2.localNode.Address() {
			numN2inProposers++
		}
	}

	passCriteria := int(maximumBlockHeight) * int(maximumRoundNumber) / numberOfNodes * 2

	if numN0inProposers >= passCriteria {
		t.Error("failed to have even number of proposers. numN0inProposers", numN0inProposers)
	}
	if numN1inProposers >= passCriteria {
		t.Error("failed to have even number of proposers. numN1inProposers", numN1inProposers)
	}
	if numN2inProposers >= passCriteria {
		t.Error("failed to have even number of proposers. numN2inProposers", numN2inProposers)
	}

	for _, nr := range nodeRunners {
		nr.Stop()
	}
}
