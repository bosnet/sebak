package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/voting"
)

// Test that ballot with empty transactions have voting.YES
func TestISAACBallotWithEmptyTransaction(t *testing.T) {
	conf := common.NewTestConfig()
	nr, _, _ := createNodeRunnerForTesting(1, conf, nil)

	latestBlock := nr.Consensus().LatestBlock()
	round := voting.Basis{
		Round:     0,
		Height:    latestBlock.Height,
		BlockHash: latestBlock.Hash,
		TotalTxs:  latestBlock.TotalTxs,
	}

	b := ballot.NewBallot(nr.localNode.Address(), nr.localNode.Address(), round, []string{})
	require.Equal(t, b.B.Vote, voting.YES)
}

// Test that the voting process ends normally with a ballot with an empty transaction.
func TestISAACBallotWithEmptyTransactionVoting(t *testing.T) {
	conf := common.NewTestConfig()
	nr, nodes, _ := createNodeRunnerForTesting(5, conf, nil)

	// `nodeRunner` is proposer's runner
	proposer := nr.localNode

	latestBlock := nr.Consensus().LatestBlock()
	require.Equal(t, uint64(1), latestBlock.Height)
	require.Equal(t, uint64(1), latestBlock.TotalTxs)

	// Generate proposed ballot in nr
	_, err := nr.proposeNewBallot(0)
	require.NoError(t, err)

	round := voting.Basis{
		Round:     0,
		Height:    latestBlock.Height,
		BlockHash: latestBlock.Hash,
		TotalTxs:  latestBlock.TotalTxs,
	}

	b := ballot.NewBallot(nr.localNode.Address(), nr.localNode.Address(), round, []string{})
	b.SetVote(ballot.StateINIT, voting.YES)

	ballotSIGN1 := GenerateEmptyTxBallot(proposer, round, ballot.StateSIGN, nodes[1], conf)
	err = ReceiveBallot(nr, ballotSIGN1)
	require.NoError(t, err)

	ballotSIGN2 := GenerateEmptyTxBallot(proposer, round, ballot.StateSIGN, nodes[2], conf)
	err = ReceiveBallot(nr, ballotSIGN2)
	require.NoError(t, err)

	ballotSIGN3 := GenerateEmptyTxBallot(proposer, round, ballot.StateSIGN, nodes[3], conf)
	err = ReceiveBallot(nr, ballotSIGN3)
	require.NoError(t, err)

	runningRounds := nr.Consensus().RunningRounds

	// Check that the transaction is in RunningRounds
	rr := runningRounds[round.Index()]
	require.NotNil(t, rr)

	result := rr.Voted[proposer.Address()].GetResult(ballot.StateSIGN)
	require.Equal(t, 3, len(result))

	ballotACCEPT1 := GenerateEmptyTxBallot(proposer, round, ballot.StateACCEPT, nodes[1], conf)
	err = ReceiveBallot(nr, ballotACCEPT1)
	require.NoError(t, err)

	ballotACCEPT2 := GenerateEmptyTxBallot(proposer, round, ballot.StateACCEPT, nodes[2], conf)
	err = ReceiveBallot(nr, ballotACCEPT2)
	require.NoError(t, err)

	ballotACCEPT3 := GenerateEmptyTxBallot(proposer, round, ballot.StateACCEPT, nodes[3], conf)
	err = ReceiveBallot(nr, ballotACCEPT3)
	require.NoError(t, err)

	ballotACCEPT4 := GenerateEmptyTxBallot(proposer, round, ballot.StateACCEPT, nodes[4], conf)
	err = ReceiveBallot(nr, ballotACCEPT4)
	require.EqualError(t, err, "ballot got consensus and will be stored")

	latestBlock = nr.Consensus().LatestBlock()
	require.Equal(t, proposer.Address(), latestBlock.Proposer)
	require.Equal(t, uint64(2), latestBlock.Height)
	require.Equal(t, uint64(2), latestBlock.TotalTxs)
	require.Equal(t, 0, len(latestBlock.Transactions))
}
