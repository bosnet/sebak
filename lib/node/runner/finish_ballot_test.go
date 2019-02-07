package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
)

func createNodeRunnerForTestingWithFileStorage(n int, conf common.Config, recv chan struct{}) (*NodeRunner, []*node.LocalNode, string) {
	var ns []*network.MemoryNetwork
	var net *network.MemoryNetwork
	var nodes []*node.LocalNode
	for i := 0; i < n; i++ {
		s, v := network.CreateMemoryNetwork(net)
		net = s
		ns = append(ns, s)
		nodes = append(nodes, v)
	}

	for j := 0; j < n; j++ {
		nodes[0].AddValidators(nodes[j].ConvertToValidator())
	}

	localNode := nodes[0]
	policy, _ := consensus.NewDefaultVotingThresholdPolicy(67)

	connectionManager := NewTestConnectionManager(
		localNode,
		ns[0],
		policy,
		recv,
	)

	st := &storage.LevelDBBackend{}
	dir, err := ioutil.TempDir("", "sebak-test")
	if err != nil {
		panic(err)
	}

	{
		var err error
		config, _ := storage.NewConfigFromString(fmt.Sprintf("file://%s", dir))

		if err = st.Init(config); err != nil {
			panic(err)
		}

		block.MakeTestBlockchain(st)
	}

	is, _ := consensus.NewISAAC(localNode, policy, connectionManager, st, conf, nil)
	is.SetProposerSelector(FixedSelector{localNode.Address()})

	tp := transaction.NewPool(conf)

	nr, err := NewNodeRunner(localNode, policy, ns[0], is, st, tp, conf)
	if err != nil {
		panic(err)
	}
	nr.isaacStateManager.blockTimeBuffer = 0

	return nr, nodes, dir
}

func testFinishBallot(withBatch bool, numberOfTransactions, numberOfOperations int) error {
	conf := common.NewTestConfig()
	nr, localNodes, dir := createNodeRunnerForTestingWithFileStorage(1, conf, nil)
	defer func() {
		nr.Storage().Close()
		os.RemoveAll(dir)
	}()

	proposerNode := localNodes[0]
	nr.Consensus().SetProposerSelector(FixedSelector{proposerNode.Address()})

	genesisBlock := block.GetGenesis(nr.Storage())
	commonAccount, _ := GetCommonAccount(nr.Storage())
	initialBalance, _ := GetGenesisBalance(nr.Storage())

	var blt *ballot.Ballot
	{
		var txs []transaction.Transaction
		var txHashes []string

		rd := voting.Basis{
			Round:     0,
			Height:    genesisBlock.Height,
			BlockHash: genesisBlock.Hash,
			TotalTxs:  genesisBlock.TotalTxs,
		}

		for i := 0; i < numberOfTransactions; i++ {
			kpA := keypair.Random()
			accountA := block.NewBlockAccount(kpA.Address(), common.Amount(common.BaseReserve))
			accountA.MustSave(nr.Storage())

			kpB := keypair.Random()
			tx := transaction.MakeTransactionCreateAccount(conf.NetworkID, kpA, kpB.Address(), common.Amount(1))

			var ops []operation.Operation
			for j := 0; j < numberOfOperations-1; j++ {
				kpC := keypair.Random()

				opb := operation.NewCreateAccount(kpC.Address(), common.Amount(1), "")
				op := operation.Operation{
					H: operation.Header{
						Type: operation.TypeCreateAccount,
					},
					B: opb,
				}
				ops = append(ops, op)
			}
			tx.B.Operations = append(tx.B.Operations, ops...)
			tx.B.SequenceID = accountA.SequenceID
			tx.Sign(kpA, conf.NetworkID)

			txHashes = append(txHashes, tx.GetHash())
			txs = append(txs, tx)
			nr.TransactionPool.Add(tx)
		}

		blt = ballot.NewBallot(proposerNode.Address(), proposerNode.Address(), rd, txHashes)

		opc, _ := ballot.NewCollectTxFeeFromBallot(*blt, commonAccount.Address, txs...)
		opi, _ := ballot.NewInflationFromBallot(*blt, commonAccount.Address, initialBalance)
		ptx, _ := ballot.NewProposerTransactionFromBallot(*blt, opc, opi)

		blt.SetProposerTransaction(ptx)
		blt.SetVote(ballot.StateINIT, voting.YES)
		blt.Sign(proposerNode.Keypair(), conf.NetworkID)
	}

	_, _, err := finishBallot(
		nr,
		*blt,
		nr.Log(),
	)

	return err
}

func TestFinishBallot(t *testing.T) {
	var err error

	err = testFinishBallot(false, 100, 100)
	require.NoError(t, err)

	err = testFinishBallot(true, 100, 100)
	require.NoError(t, err)
}
