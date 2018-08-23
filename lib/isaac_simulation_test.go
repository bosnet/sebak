package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
)

func TestIsaacSimulationProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(5)

	tx, txByte := getTransaction(t)

	message := sebaknetwork.Message{Type: sebaknetwork.MessageFromClient, Data: txByte}

	nodeRunner := nodeRunners[0]

	// `nodeRunner` is proposer's runner
	nodeRunner.SetProposerCalculator(SelfProposerCalculator{})
	proposer := nodeRunner.localNode

	nodeRunner.Consensus().SetLatestConsensusedBlock(genesisBlock)

	var err error
	err = nodeRunner.handleMessageFromClient(message)

	require.Nil(t, err)
	require.True(t, nodeRunner.Consensus().TransactionPool.Has(tx.GetHash()))

	// Generate proposed ballot in nodeRunner
	roundNumber := uint64(0)
	err = nodeRunner.proposeNewBallot(roundNumber)
	require.Nil(t, err)
	round := Round{
		Number:      roundNumber,
		BlockHeight: nodeRunner.Consensus().LatestConfirmedBlock.Height,
		BlockHash:   nodeRunner.Consensus().LatestConfirmedBlock.Hash,
		TotalTxs:    nodeRunner.Consensus().LatestConfirmedBlock.TotalTxs,
	}
	runningRounds := nodeRunner.Consensus().RunningRounds

	// Check that the transaction is in RunningRounds
	rr := runningRounds[round.Hash()]
	txHashs := rr.Transactions[proposer.Address()]
	require.Equal(t, tx.GetHash(), txHashs[0])

	ballotSIGN1 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateSIGN, nodeRunners[1].localNode)
	err = receiveBallot(t, nodeRunner, ballotSIGN1)
	require.Nil(t, err)

	ballotSIGN2 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateSIGN, nodeRunners[2].localNode)
	err = receiveBallot(t, nodeRunner, ballotSIGN2)
	require.Nil(t, err)

	ballotSIGN3 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateSIGN, nodeRunners[3].localNode)
	err = receiveBallot(t, nodeRunner, ballotSIGN3)
	require.Nil(t, err)

	ballotSIGN4 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateSIGN, nodeRunners[4].localNode)
	err = receiveBallot(t, nodeRunner, ballotSIGN4)
	require.Nil(t, err)

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(sebakcommon.BallotStateSIGN)))

	ballotACCEPT1 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateACCEPT, nodeRunners[1].localNode)
	err = receiveBallot(t, nodeRunner, ballotACCEPT1)
	require.Nil(t, err)

	ballotACCEPT2 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateACCEPT, nodeRunners[2].localNode)
	err = receiveBallot(t, nodeRunner, ballotACCEPT2)
	require.Nil(t, err)

	ballotACCEPT3 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateACCEPT, nodeRunners[3].localNode)
	err = receiveBallot(t, nodeRunner, ballotACCEPT3)
	require.Nil(t, err)

	ballotACCEPT4 := generateBallot(t, proposer, round, tx, sebakcommon.BallotStateACCEPT, nodeRunners[4].localNode)
	err = receiveBallot(t, nodeRunner, ballotACCEPT4)
	require.EqualError(t, err, "stop checker and return: ballot got consensus and will be stored")

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(sebakcommon.BallotStateACCEPT)))

	block := nodeRunner.Consensus().LatestConfirmedBlock
	require.Equal(t, proposer.Address(), block.Proposer)
	require.Equal(t, 1, len(block.Transactions))
	require.Equal(t, tx.GetHash(), block.Transactions[0])
}

func getTransaction(t *testing.T) (tx Transaction, txByte []byte) {
	initialBalance := sebakcommon.Amount(1)
	kpNewAccount, _ := keypair.Random()

	tx = makeTransactionCreateAccount(kp, kpNewAccount.Address(), initialBalance)
	tx.B.Checkpoint = account.Checkpoint
	tx.Sign(kp, networkID)

	var err error

	txByte, err = tx.Serialize()
	require.Nil(t, err)

	return
}

func generateBallot(t *testing.T, proposer *sebaknode.LocalNode, round Round, tx Transaction, ballotState sebakcommon.BallotState, sender *sebaknode.LocalNode) *Ballot {
	ballot := NewBallot(proposer, round, []string{tx.GetHash()})
	ballot.SetVote(sebakcommon.BallotStateINIT, VotingYES)
	ballot.Sign(proposer.Keypair(), networkID)

	ballot.SetSource(sender.Address())
	ballot.SetVote(ballotState, VotingYES)
	ballot.Sign(sender.Keypair(), networkID)

	err := ballot.IsWellFormed(networkID)
	require.Nil(t, err)

	return ballot
}

func receiveBallot(t *testing.T, nodeRunner *NodeRunner, ballot *Ballot) error {
	data, err := ballot.Serialize()
	require.Nil(t, err)

	ballotMessage := sebaknetwork.Message{Type: sebaknetwork.BallotMessage, Data: data}
	err = nodeRunner.handleBallotMessage(ballotMessage)
	return err
}
