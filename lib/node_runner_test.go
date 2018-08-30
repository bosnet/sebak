package sebak

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"
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
	tlsKey *sebaknetwork.KeyGenerator
)

func init() {
	kp, _ = keypair.Random()

	dir, err := ioutil.TempDir("/tmp/", "sebak-test")
	if err != nil {
		panic(err)
	}

	tlsKey = sebaknetwork.NewKeyGenerator(dir, "sebak-test.crt", "sebak-test.key")
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
		is, _ := NewISAAC(networkID, v, p)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()

		account.Save(st)
		genesisBlock = MakeGenesisBlock(st, *account)

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

func createTestNodeRunnersHTTP2Network(n int) (nodeRunners []*NodeRunner, rootKP *keypair.Full) {
	var nodes []*sebaknode.LocalNode
	var ports []int
	for i := 0; i < n; i++ {
		kp, _ := keypair.Random()
		port := sebakcommon.GetFreePort(ports...)
		if port < 1 {
			panic("failed to find free port")
		}
		ports = append(ports, port)

		endpoint, _ := sebakcommon.NewEndpointFromString(
			fmt.Sprintf(
				"http://localhost:%d?NodeName=%s&HTTP2LogOutput=%s",
				port,
				kp.Address(),
				"/dev/null",
			),
		)
		node, _ := sebaknode.NewLocalNode(kp, endpoint, "")
		nodes = append(nodes, node)
	}

	for i, node0 := range nodes {
		for j, node1 := range nodes {
			if i == j {
				continue
			}
			node0.AddValidators(node1.ConvertToValidator())
		}
	}

	rootKP, _ = keypair.Random()
	genesisAccount := block.NewBlockAccount(
		rootKP.Address(),
		10000000000000,
		sebakcommon.MakeGenesisCheckpoint(networkID),
	)
	for _, node := range nodes {
		vth, _ := NewDefaultVotingThresholdPolicy(66, 66)
		is, _ := NewISAAC(networkID, node, vth)
		st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
		networkConfig, _ := sebaknetwork.NewHTTP2NetworkConfigFromEndpoint(node.Endpoint())
		network := sebaknetwork.NewHTTP2Network(networkConfig)
		nodeRunner, _ := NewNodeRunner(string(networkID), node, vth, network, is, st)

		genesisAccount.Save(nodeRunner.Storage())
		MakeGenesisBlock(st, *genesisAccount)

		nodeRunners = append(nodeRunners, nodeRunner)
	}

	return nodeRunners, rootKP
}

func createTestNodeRunnersHTTP2NetworkWithReady(n int) (nodeRunners []*NodeRunner, rootKP *keypair.Full) {
	nodeRunners, rootKP = createTestNodeRunnersHTTP2Network(n)

	for _, nr := range nodeRunners {
		go func(nodeRunner *NodeRunner) {
			if err := nodeRunner.Start(); err != nil {
				panic(err)
			}
		}(nr)
	}

	return

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

	return
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

func TestNodeRunnerSaveBlock(t *testing.T) {
	numberOfNodes := 4
	nodeRunners, _ := createTestNodeRunnersHTTP2NetworkWithReady(numberOfNodes)

	previousBlockHeight := map[string]uint64{}
	for _, nodeRunner := range nodeRunners {
		bck, err := GetLatestBlock(nodeRunner.Storage())
		if err != nil {
			t.Error(err)
			return
		}
		previousBlockHeight[nodeRunner.Node().Address()] = bck.Height
	}

	var wg sync.WaitGroup
	wg.Add(numberOfNodes)
	checkerDeferFunc := func(n int, checker sebakcommon.Checker, err error) {
		if _, ok := err.(sebakcommon.CheckerStop); !ok {
			return
		}

		wg.Done()
	}

	for _, nodeRunner := range nodeRunners {
		nodeRunner.SetHandleMessageCheckerDeferFunc(checkerDeferFunc)
	}

	wg.Wait()

	for _, nodeRunner := range nodeRunners {
		bck, err := GetLatestBlock(nodeRunner.Storage())
		if err != nil {
			t.Error(err)
			return
		}
		previous := previousBlockHeight[nodeRunner.Node().Address()]
		if previous+1 != bck.Height {
			t.Error("nil block must be stored")
			return
		}

		if len(bck.Transactions) != 0 {
			t.Error("`Block..Transactions` must be empty")
		}
	}
}
