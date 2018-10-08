package runner

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
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

func createTestNodeRunner(n int, conf common.Config) []*NodeRunner {
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
			nodes[i].AddValidators(nodes[j].ConvertToValidator())
		}
	}

	address := kp.Address()
	balance := common.BaseFee.MustAdd(common.BaseReserve)
	account = block.NewBlockAccount(address, balance)
	var nodeRunners []*NodeRunner
	for i := 0; i < n; i++ {
		localNode := nodes[i]
		policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)

		connectionManager := network.NewValidatorConnectionManager(
			localNode,
			ns[i],
			policy,
		)

		is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager, conf)
		st := storage.NewTestStorage()

		account.Save(st)

		commonKP, _ := keypair.Random()
		commonAccount := block.NewBlockAccount(commonKP.Address(), 0)
		commonAccount.Save(st)

		genesisBlock, _ = block.MakeGenesisBlock(st, *account, *commonAccount, networkID)

		nr, err := NewNodeRunner(string(networkID), localNode, policy, ns[i], is, st, conf)
		if err != nil {
			panic(err)
		}
		nodeRunners = append(nodeRunners, nr)
	}

	return nodeRunners
}

func createTestNodeRunnerWithReady(n int) []*NodeRunner {
	nodeRunners := createTestNodeRunner(n, common.NewConfig())

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

	for _, node0 := range nodes {
		for _, node1 := range nodes {
			node0.AddValidators(node1.ConvertToValidator())
		}
	}

	rootKP, _ = keypair.Random()
	genesisAccount := block.NewBlockAccount(
		rootKP.Address(),
		common.MaximumBalance,
	)
	commonKP, _ := keypair.Random()
	commonAccount := block.NewBlockAccount(commonKP.Address(), 0)

	for _, node := range nodes {
		policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)
		st := storage.NewTestStorage()
		networkConfig, _ := network.NewHTTP2NetworkConfigFromEndpoint(node.Alias(), node.Endpoint())
		n := network.NewHTTP2Network(networkConfig)

		connectionManager := network.NewValidatorConnectionManager(
			node,
			n,
			policy,
		)

		conf := common.NewConfig()
		is, _ := consensus.NewISAAC(networkID, node, policy, connectionManager, conf)

		genesisAccount.Save(st)
		commonAccount.Save(st)

		block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)

		nodeRunner, _ := NewNodeRunner(string(networkID), node, policy, n, is, st, conf)
		nodeRunners = append(nodeRunners, nodeRunner)
	}

	return nodeRunners, rootKP
}

func createTestNodeRunnersHTTP2NetworkWithReady(n int) (nodeRunners []*NodeRunner, rootKP *keypair.Full) {
	nodeRunners, rootKP = createTestNodeRunnersHTTP2Network(n)

	for _, nr := range nodeRunners {
		go func(nodeRunner *NodeRunner) {
			if err := nodeRunner.Start(); err != nil {
				if err == http.ErrServerClosed {
					return
				}
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

// Check that createTestNodeRunner creates the appropriate number of node runners.
func TestCreateNodeRunner(t *testing.T) {
	nodeRunners := createTestNodeRunner(3, common.NewConfig())

	require.Equal(t, 3, len(nodeRunners))
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
