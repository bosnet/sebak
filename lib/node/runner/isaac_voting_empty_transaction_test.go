package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
)

// Test that ballot with empty transactions have VotingYES
func TestISAACBallotWithEmptyTransaction(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	nr, _, _ := createNodeRunnerForTesting(1, conf, nil)

	latestBlock := nr.Consensus().LatestConfirmedBlock()
	round := round.Round{
		Number:      0,
		BlockHeight: latestBlock.Height,
		BlockHash:   latestBlock.Hash,
		TotalTxs:    latestBlock.TotalTxs,
	}

	b := ballot.NewBallot(nr.localNode, round, []string{})
	require.Equal(t, b.B.Vote, ballot.VotingYES)
}

// Test that the voting process ends normally with a ballot with an empty transaction.
func TestISAACBallotWithEmptyTransactionVoting(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	nr, nodes, _ := createNodeRunnerForTesting(5, conf, nil)

	// `nodeRunner` is proposer's runner
	proposer := nr.localNode

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)
	latestBlock := nr.Consensus().LatestConfirmedBlock()
	require.Equal(t, uint64(1), latestBlock.Height)
	require.Equal(t, uint64(0), latestBlock.TotalTxs)

	// Generate proposed ballot in nr
	err := nr.proposeNewBallot(0)
	require.Nil(t, err)

	round := round.Round{
		Number:      0,
		BlockHeight: latestBlock.Height,
		BlockHash:   latestBlock.Hash,
		TotalTxs:    latestBlock.TotalTxs,
	}

	b := ballot.NewBallot(nr.localNode, round, []string{})
	b.SetVote(ballot.StateINIT, ballot.VotingYES)

	ballotSIGN1 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateSIGN, nodes[1])
	err = ReceiveBallot(t, nr, ballotSIGN1)
	require.Nil(t, err)

	ballotSIGN2 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateSIGN, nodes[2])
	err = ReceiveBallot(t, nr, ballotSIGN2)
	require.Nil(t, err)

	ballotSIGN3 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateSIGN, nodes[3])
	err = ReceiveBallot(t, nr, ballotSIGN3)
	require.Nil(t, err)

	runningRounds := nr.Consensus().RunningRounds

	// Check that the transaction is in RunningRounds
	rr := runningRounds[round.Hash()]
	require.NotNil(t, rr)

	result := rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)
	require.Equal(t, 3, len(result))

	ballotACCEPT1 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateACCEPT, nodes[1])
	err = ReceiveBallot(t, nr, ballotACCEPT1)
	require.Nil(t, err)

	ballotACCEPT2 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateACCEPT, nodes[2])
	err = ReceiveBallot(t, nr, ballotACCEPT2)
	require.Nil(t, err)

	ballotACCEPT3 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateACCEPT, nodes[3])
	err = ReceiveBallot(t, nr, ballotACCEPT3)
	require.Nil(t, err)

	ballotACCEPT4 := GenerateEmptyTxBallot(t, proposer, round, ballot.StateACCEPT, nodes[4])
	err = ReceiveBallot(t, nr, ballotACCEPT4)
	require.EqualError(t, err, "ballot got consensus and will be stored")

	require.Equal(t, 4, len(rr.Voted[proposer.Address()].GetResult(ballot.StateACCEPT)))

	lastConfirmedBlock := nr.Consensus().LatestConfirmedBlock()
	require.Equal(t, proposer.Address(), lastConfirmedBlock.Proposer)
	require.Equal(t, uint64(2), lastConfirmedBlock.Height)
	require.Equal(t, uint64(0), lastConfirmedBlock.TotalTxs)
	require.Equal(t, 0, len(lastConfirmedBlock.Transactions))
}
