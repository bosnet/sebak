package node_runner

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/storage/block"
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

	_, network, localNode := network.CreateNewMemoryNetwork()

	vth, _ := consensus.NewDefaultVotingThresholdPolicy(66, 66)
	is, _ := consensus.NewISAAC(networkID, localNode, vth)
	st, _ := storage.NewTestMemoryLevelDBBackend()
	nodeRunner, _ := NewNodeRunner(string(networkID), localNode, vth, network, is, st)
	return nodeRunner, localNode
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
