package runner

import (
	"sync"
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
	kp           *keypair.Full
	account      *block.BlockAccount
	genesisBlock block.Block
)

func init() {
	kp, _ = keypair.Random()
}

func MakeNodeRunner() (*NodeRunner, *node.LocalNode) {
	_, n, localNode := network.CreateMemoryNetwork(nil)

	policy, _ := consensus.NewDefaultVotingThresholdPolicy(66, 66)

	connectionManager := network.NewConnectionManager(
		localNode,
		n,
		policy,
		localNode.GetValidators(),
	)
	connectionManager.SetProposerCalculator(network.NewSimpleProposerCalculator(connectionManager))

	is, _ := consensus.NewISAAC(networkID, localNode, policy, connectionManager)
	st, _ := storage.NewTestMemoryLevelDBBackend()
	conf := consensus.NewISAACConfiguration()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, policy, n, is, st, conf)
	return nodeRunner, localNode
}

func GetTransaction(t *testing.T) (tx transaction.Transaction, txByte []byte) {
	initialBalance := common.Amount(1)
	kpNewAccount, _ := keypair.Random()

	tx = transaction.MakeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.SequenceID = account.SequenceID
	tx.Sign(kp, networkID)

	var err error

	txByte, err = tx.Serialize()
	require.Nil(t, err)

	return
}

func TestGenerateNewSequenceID() uint64 {
	return 0
}

type SelfProposerCalculator struct {
	nodeRunner *NodeRunner
}

func (c SelfProposerCalculator) Calculate(_ uint64, _ uint64) string {
	return c.nodeRunner.localNode.Address()
}

type TheOtherProposerCalculator struct {
	nodeRunner *NodeRunner
}

func (c TheOtherProposerCalculator) Calculate(_ uint64, _ uint64) string {
	for _, v := range c.nodeRunner.ConnectionManager().AllValidators() {
		if v != c.nodeRunner.localNode.Address() {
			return v
		}
	}
	panic("There is no the other validators")
}

type SelfProposerThenNotProposer struct {
	nodeRunner *NodeRunner
}

func (c *SelfProposerThenNotProposer) Calculate(blockHeight uint64, roundNumber uint64) string {
	if blockHeight < 2 && roundNumber == 0 {
		return c.nodeRunner.localNode.Address()
	} else {
		for _, v := range c.nodeRunner.ConnectionManager().AllValidators() {
			if v != c.nodeRunner.localNode.Address() {
				return v
			}
		}
		panic("There is no the other validators")
	}
}

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx transaction.Transaction, ballotState ballot.State, sender *node.LocalNode) *block.Ballot {
	b := block.NewBallot(proposer, round, []string{tx.GetHash()})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)
	b.Sign(proposer.Keypair(), networkID)

	b.SetSource(sender.Address())
	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	err := b.IsWellFormed(networkID)
	require.Nil(t, err)

	return b
}

func GenerateEmptyTxBallot(t *testing.T, proposer *node.LocalNode, round round.Round, ballotState ballot.State, sender *node.LocalNode) *block.Ballot {
	b := block.NewBallot(proposer, round, []string{})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)
	b.Sign(proposer.Keypair(), networkID)

	b.SetSource(sender.Address())
	b.SetVote(ballotState, ballot.VotingYES)
	b.Sign(sender.Keypair(), networkID)

	err := b.IsWellFormed(networkID)
	require.Nil(t, err)

	return b
}

func ReceiveBallot(t *testing.T, nodeRunner *NodeRunner, ballot *block.Ballot) error {
	data, err := ballot.Serialize()
	require.Nil(t, err)

	ballotMessage := common.NetworkMessage{Type: common.BallotMessage, Data: data}
	err = nodeRunner.handleBallotMessage(ballotMessage)
	return err
}

type TestBroadcaster struct {
	sync.RWMutex
	messages []common.Message
	recv     chan struct{}
}

func NewTestBroadcaster(r chan struct{}) *TestBroadcaster {
	p := &TestBroadcaster{}
	p.messages = []common.Message{}
	p.recv = r
	return p
}

func (b *TestBroadcaster) Broadcast(message common.Message, _ func(string, error)) {
	b.Lock()
	defer b.Unlock()
	b.messages = append(b.messages, message)
	if b.recv != nil {
		b.recv <- struct{}{}
	}
	return
}

func (b *TestBroadcaster) Messages() []common.Message {
	b.RLock()
	defer b.RUnlock()
	messages := make([]common.Message, len(b.messages))
	copy(messages, b.messages)
	return messages
}
