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
	kp             *keypair.Full
	account        *block.BlockAccount
	genesisBlock   block.Block
	initialBalance common.Amount
	commonKP       *keypair.Full
	commonAccount  *block.BlockAccount
)

func init() {
	kp, _ = keypair.Random()
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
	st := storage.NewTestStorage()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, policy, n, is, st, conf)
	return nodeRunner, localNode
}

func GetTransaction(t *testing.T) (tx transaction.Transaction, txByte []byte) {
	initialBalance := common.Amount(common.BaseReserve)
	kpNewAccount, _ := keypair.Random()

	tx = transaction.MakeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.SequenceID = uint64(0)
	tx.Sign(kp, networkID)

	var err error

	txByte, err = tx.Serialize()
	require.Nil(t, err)

	return
}

func TestGenerateNewSequenceID() uint64 {
	return 0
}

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx transaction.Transaction, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(proposer.Address(), round, []string{tx.GetHash()})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)

	opi, _ := ballot.NewInflationFromBallot(*b, commonAccount.Address, initialBalance)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, commonAccount.Address, tx)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)

	b.Sign(proposer.Keypair(), networkID)

	b.SetSource(sender.Address())
	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	err := b.IsWellFormed(networkID, conf)
	require.Nil(t, err)

	return b
}

func GenerateEmptyTxBallot(t *testing.T, proposer *node.LocalNode, round round.Round, ballotState ballot.State, sender *node.LocalNode, conf common.Config) *ballot.Ballot {
	b := ballot.NewBallot(proposer.Address(), round, []string{})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)

	opi, _ := ballot.NewInflationFromBallot(*b, commonAccount.Address, initialBalance)
	opc, _ := ballot.NewCollectTxFeeFromBallot(*b, commonAccount.Address)
	ptx, _ := ballot.NewProposerTransactionFromBallot(*b, opc, opi)
	b.SetProposerTransaction(ptx)

	b.Sign(proposer.Keypair(), networkID)

	b.SetSource(sender.Address())
	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	err := b.IsWellFormed(networkID, conf)
	require.Nil(t, err)

	return b
}

func ReceiveBallot(t *testing.T, nodeRunner *NodeRunner, ballot *ballot.Ballot) error {
	data, err := ballot.Serialize()
	require.Nil(t, err)

	ballotMessage := common.NetworkMessage{Type: common.BallotMessage, Data: data}
	err = nodeRunner.handleBallotMessage(ballotMessage)
	return err
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

	st := storage.NewTestStorage()

	balance := common.MaximumBalance
	genesisAccount := block.NewBlockAccount(kp.Address(), balance)
	genesisAccount.Save(st)

	commonKP, _ = keypair.Random()
	commonAccount = block.NewBlockAccount(commonKP.Address(), 0)
	commonAccount.Save(st)

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

	genesisBlock, _ = block.MakeGenesisBlock(st, *genesisAccount, *commonAccount, networkID)
	initialBalance = balance

	nr, err := NewNodeRunner(string(networkID), localNode, policy, ns[0], is, st, conf)
	if err != nil {
		panic(err)
	}
	nr.isaacStateManager.blockTimeBuffer = 0

	return nr, nodes, connectionManager
}
