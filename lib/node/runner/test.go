package runner

import (
	"sync"
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

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

func MakeNodeRunner(prev *network.MemoryNetwork) (*NodeRunner, *node.LocalNode) {
	_, network, localNode := network.CreateMemoryNetwork(prev)

	vth, _ := consensus.NewDefaultVotingThresholdPolicy(66, 66)
	is, _ := consensus.NewISAAC(networkID, localNode, vth)
	st, _ := storage.NewTestMemoryLevelDBBackend()
	conf := consensus.NewISAACConfiguration()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, vth, network, is, st, conf)
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
}

func (c SelfProposerCalculator) Calculate(nr *NodeRunner, _ uint64, _ uint64) string {
	return nr.localNode.Address()
}

type TheOtherProposerCalculator struct {
}

func (c TheOtherProposerCalculator) Calculate(nr *NodeRunner, _ uint64, _ uint64) string {
	for _, v := range nr.ConnectionManager().AllValidators() {
		if v != nr.localNode.Address() {
			return v
		}
	}
	panic("There is no the other validators")
}

type SelfProposerThenNotProposer struct {
}

func (c *SelfProposerThenNotProposer) Calculate(nr *NodeRunner, blockHeight uint64, roundNumber uint64) string {
	if blockHeight < 2 && roundNumber == 0 {
		return nr.localNode.Address()
	} else {
		for _, v := range nr.ConnectionManager().AllValidators() {
			if v != nr.localNode.Address() {
				return v
			}
		}
		panic("There is no the other validators")
	}
}

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx transaction.Transaction, ballotState common.BallotState, sender *node.LocalNode) *block.Ballot {
	ballot := block.NewBallot(proposer, round, []string{tx.GetHash()})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func GenerateEmptyTxBallot(t *testing.T, proposer *node.LocalNode, round round.Round, ballotState common.BallotState, sender *node.LocalNode) *block.Ballot {
	ballot := block.NewBallot(proposer, round, []string{})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
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
