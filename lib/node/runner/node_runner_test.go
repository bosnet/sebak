package runner

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/transaction"
	"github.com/stellar/go/keypair"
)

var (
	tlsKey *network.KeyGenerator
)

func init() {
	kp, _ = keypair.Random()

	dir, err := ioutil.TempDir("/tmp/", "sebak-test")
	if err != nil {
		panic(err)
	}

	tlsKey = network.NewKeyGenerator(dir, "sebak-test.crt", "sebak-test.key")
}

func createTestNodeRunner(n int) []*NodeRunner {
	var ns []*network.MemoryNetwork
	var net *network.MemoryNetwork
	var nodes []*node.LocalNode
	for i := 0; i < n; i++ {
		_, s, v := network.CreateMemoryNetwork(net)
		net = s
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

	address := kp.Address()
	balance := common.BaseFee.MustAdd(1)
	account = block.NewBlockAccount(address, balance)
	var nodeRunners []*NodeRunner
	for i := 0; i < n; i++ {
		v := nodes[i]
		p, _ := consensus.NewDefaultVotingThresholdPolicy(66, 66)
		is, _ := consensus.NewISAAC(networkID, v, p)
		st, _ := storage.NewTestMemoryLevelDBBackend()

		account.Save(st)
		genesisBlock = block.MakeGenesisBlock(st, *account)

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
	var nodes []*node.LocalNode
	var ports []int
	for i := 0; i < n; i++ {
		kp, _ := keypair.Random()
		port := common.GetFreePort(ports...)
		if port < 1 {
			panic("failed to find free port")
		}
		ports = append(ports, port)

		endpoint, _ := common.NewEndpointFromString(
			fmt.Sprintf(
				"http://localhost:%d?NodeName=%s",
				port,
				kp.Address(),
			),
		)
		node, _ := node.NewLocalNode(kp, endpoint, "")
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
	)
	for _, node := range nodes {
		vth, _ := consensus.NewDefaultVotingThresholdPolicy(66, 66)
		is, _ := consensus.NewISAAC(networkID, node, vth)
		st, _ := storage.NewTestMemoryLevelDBBackend()
		networkConfig, _ := network.NewHTTP2NetworkConfigFromEndpoint(node.Alias(), node.Endpoint())
		network := network.NewHTTP2Network(networkConfig)
		nodeRunner, _ := NewNodeRunner(string(networkID), node, vth, network, is, st)

		genesisAccount.Save(nodeRunner.Storage())
		block.MakeGenesisBlock(st, *genesisAccount)

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
	numberOfNodes := 3
	nodeRunners := createTestNodeRunnerWithReady(numberOfNodes)

	kpNewAccount, _ := keypair.Random()

	nr0 := nodeRunners[0]

	client := nr0.Network().GetClient(nr0.Node().Endpoint())

	initialBalance := common.Amount(1)
	tx := transaction.MakeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.SequenceID = account.SequenceID
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

/*
func TestNodeRunnerSaveBlock(t *testing.T) {
	numberOfNodes := 4
	nodeRunners, _ := createTestNodeRunnersHTTP2NetworkWithReady(numberOfNodes)
	previousBlockHeight := map[string]uint64{}
	for _, nodeRunner := range nodeRunners {
		bck, err := block.GetLatestBlock(nodeRunner.Storage())
		if err != nil {
			t.Error(err)
			return
		}
		previousBlockHeight[nodeRunner.Node().Address()] = bck.Height
	}
	var wg sync.WaitGroup
	wg.Add(numberOfNodes)
	checkerDeferFunc := func(n int, checker common.Checker, err error) {
		if _, ok := err.(common.CheckerStop); !ok {
			return
		}
		wg.Done()
	}
	for _, nodeRunner := range nodeRunners {
		nodeRunner.SetHandleMessageCheckerDeferFunc(checkerDeferFunc)
	}
	wg.Wait()
	for _, nodeRunner := range nodeRunners {
		bck, err := block.GetLatestBlock(nodeRunner.Storage())
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
*/
