package runner

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-test-network")

var (
	genesisKP     *keypair.Full
	commonKP      *keypair.Full
	commonAccount *block.BlockAccount
)

func init() {
	var err error
	genesisKP, err = keypair.Random()
	if err != nil {
		panic(err)
	}
	commonKP, err = keypair.Random()
	if err != nil {
		panic(err)
	}
}

func MakeNodeRunner() (*NodeRunner, *node.LocalNode) {
	_, n, localNode := network.CreateMemoryNetwork(nil)

	policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)

	localNode.AddValidators(localNode.ConvertToValidator())
	connectionManager := network.NewValidatorConnectionManager(
		localNode,
		n,
		policy,
	)

	conf := common.NewConfig()
	is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager, conf)
	st := InitTestBlockchain()

	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, policy, n, is, st, conf)
	return nodeRunner, localNode
}

func GetTransaction() (transaction.Transaction, []byte) {
	kpNewAccount, _ := keypair.Random()

	tx := transaction.MakeTransactionCreateAccount(genesisKP, kpNewAccount.Address(), common.BaseReserve)
	tx.B.SequenceID = uint64(0)
	tx.Sign(genesisKP, networkID)

	if txByte, err := tx.Serialize(); err != nil {
		panic(err)
	} else {
		return tx, txByte
	}
}

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx transaction.Transaction, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(sender.Address(), proposer.Address(), round, []string{tx.GetHash()})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)

	opi, _ := ballot.NewInflationFromBallot(*b, commonAccount.Address, common.BaseReserve)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, commonAccount.Address, tx)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)
	b.Sign(proposer.Keypair(), networkID)

	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	err := b.IsWellFormed(networkID, conf)
	require.Nil(t, err)

	return b
}

func GenerateEmptyTxBallot(proposer *node.LocalNode, round round.Round, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(sender.Address(), proposer.Address(), round, []string{})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)

	opi, _ := ballot.NewInflationFromBallot(*b, commonAccount.Address, common.BaseReserve)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, commonAccount.Address)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)
	b.Sign(proposer.Keypair(), networkID)

	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	if err := b.IsWellFormed(networkID, conf); err != nil {
		panic(err)
	}

	return b
}

func ReceiveBallot(nodeRunner *NodeRunner, ballot *ballot.Ballot) error {
	data, err := ballot.Serialize()
	if err != nil {
		panic(err)
	}

	ballotMessage := common.NetworkMessage{Type: common.BallotMessage, Data: data}
	return nodeRunner.handleBallotMessage(ballotMessage)
}

func createNodeRunnerForTesting(n int, conf common.Config, recv chan struct{}) (*NodeRunner, []*node.LocalNode, *TestConnectionManager) {
	var ns []*network.MemoryNetwork
	var net *network.MemoryNetwork
	var nodes []*node.LocalNode
	for i := 0; i < n; i++ {
		_, s, v := network.CreateMemoryNetwork(net)
		net = s
		ns = append(ns, s)
		nodes = append(nodes, v)
	}

	for j := 0; j < n; j++ {
		nodes[0].AddValidators(nodes[j].ConvertToValidator())
	}

	localNode := nodes[0]
	policy, _ := consensus.NewDefaultVotingThresholdPolicy(66)

	connectionManager := NewTestConnectionManager(
		localNode,
		ns[0],
		policy,
		recv,
	)

	is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager, common.NewConfig())
	is.SetProposerSelector(FixedSelector{localNode.Address()})

	st := InitTestBlockchain()
	nr, err := NewNodeRunner(string(networkID), localNode, policy, ns[0], is, st, conf)
	if err != nil {
		panic(err)
	}
	nr.isaacStateManager.blockTimeBuffer = 0
	genesisBlock := block.GetGenesis(st)
	nr.Consensus().SetLatestBlock(genesisBlock)

	return nr, nodes, connectionManager
}

// Initialize the blockchain with a genesis account and a common account
func InitTestBlockchain() *storage.LevelDBBackend {
	st := storage.NewTestStorage()

	balance := common.MaximumBalance
	genesisAccount := block.NewBlockAccount(genesisKP.Address(), balance)
	genesisAccount.Save(st)

	commonKP, _ = keypair.Random()
	commonAccount = block.NewBlockAccount(commonKP.Address(), 0)
	commonAccount.Save(st)

	genesisBlock, _ := block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)
	genesisBlock.Save(st)

	return st
}
