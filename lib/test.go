package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

var networkID []byte = []byte("sebak-test-network")

var (
	kp           *keypair.Full
	account      *block.BlockAccount
	genesisBlock Block
)

func init() {
	kp, _ = keypair.Random()
}

func MakeNodeRunner() (*NodeRunner, *node.LocalNode) {

	_, network, localNode := network.CreateNewMemoryNetwork()

	vth, _ := NewDefaultVotingThresholdPolicy(66, 66)
	is, _ := NewISAAC(networkID, localNode, vth)
	st, _ := storage.NewTestMemoryLevelDBBackend()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, vth, network, is, st)
	return nodeRunner, localNode
}

func testMakeNewBlock(transactions []string) Block {
	kp, _ := keypair.Random()

	return NewBlock(
		kp.Address(),
		round.Round{
			BlockHeight: 0,
			BlockHash:   "",
		},
		transactions,
		common.NowISO8601(),
	)
}

func TestMakeNewBlockOperation(networkID []byte, n int) (bos []BlockOperation) {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	for _, op := range tx.B.Operations {
		bos = append(bos, NewBlockOperationFromOperation(op, tx))
	}

	return
}

func TestMakeNewBlockTransaction(networkID []byte, n int) BlockTransaction {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	block := testMakeNewBlock([]string{tx.GetHash()})
	a, _ := tx.Serialize()
	return NewBlockTransactionFromTransaction(block.Hash, tx, a)
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

func GenerateBallot(t *testing.T, proposer *node.LocalNode, round round.Round, tx transaction.Transaction, ballotState common.BallotState, sender *node.LocalNode) *Ballot {
	ballot := NewBallot(proposer, round, []string{tx.GetHash()})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func GenerateEmptyTxBallot(t *testing.T, proposer *node.LocalNode, round round.Round, ballotState common.BallotState, sender *node.LocalNode) *Ballot {
	ballot := NewBallot(proposer, round, []string{})
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, common.VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func ReceiveBallot(t *testing.T, nodeRunner *NodeRunner, ballot *Ballot) error {
	data, err := ballot.Serialize()
	require.Nil(t, err)

	ballotMessage := common.NetworkMessage{Type: common.BallotMessage, Data: data}
	err = nodeRunner.handleBallotMessage(ballotMessage)
	return err
}
