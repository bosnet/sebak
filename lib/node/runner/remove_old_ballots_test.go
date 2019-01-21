package runner

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/voting"
	"github.com/stretchr/testify/require"
)

func insertOldRunningRound(t *testing.T, is *consensus.ISAAC, blt ballot.Ballot) {
	if runningRound, found := is.RunningRounds[blt.VotingBasis().Index()]; !found {
		rr, err := consensus.NewRunningRound(blt.Proposer(), blt)
		require.NoError(t, err)
		is.RunningRounds[blt.VotingBasis().Index()] = rr
	} else {
		runningRound.Vote(blt)
	}
}

func TestRemoveOldBallots(t *testing.T) {
	nr, nodes, _ := createNodeRunnerForTesting(3, common.NewTestConfig(), nil)

	basis := voting.Basis{
		Height: 10,
		Round:  0,
	}

	is := nr.consensus

	invalidBlt := *ballot.NewBallot(nodes[0].Address(), nodes[0].Address(), basis, []string{})
	invalidBlt.SetVote(ballot.StateSIGN, voting.YES)
	invalidBlt.B.Proposed.Confirmed = common.FormatISO8601(time.Now().Add(-2 * time.Minute))
	invalidBlt.B.Confirmed = common.FormatISO8601(time.Now().Add(-3 * time.Minute))

	proposedInvalidBlt := *ballot.NewBallot(nodes[1].Address(), nodes[1].Address(), basis, []string{})
	proposedInvalidBlt.SetVote(ballot.StateSIGN, voting.YES)
	proposedInvalidBlt.B.Proposed.Confirmed = common.FormatISO8601(time.Now().Add(-2 * time.Minute))
	proposedInvalidBlt.B.Confirmed = common.FormatISO8601(time.Now())

	validBlt := *ballot.NewBallot(nodes[2].Address(), nodes[2].Address(), basis, []string{})
	validBlt.SetVote(ballot.StateSIGN, voting.YES)
	validBlt.B.Proposed.Confirmed = common.FormatISO8601(time.Now())
	validBlt.B.Confirmed = common.FormatISO8601(time.Now())

	insertOldRunningRound(t, is, invalidBlt)
	insertOldRunningRound(t, is, proposedInvalidBlt)
	insertOldRunningRound(t, is, validBlt)

	rr := is.RunningRounds[basis.Index()]
	require.True(t, rr.IsVoted(invalidBlt))
	require.True(t, rr.IsVoted(proposedInvalidBlt))
	require.True(t, rr.IsVoted(validBlt))

	needRenewal := is.RemoveOldBallots(invalidBlt)
	require.True(t, needRenewal)

	require.False(t, rr.IsVoted(invalidBlt))
	require.True(t, rr.IsVoted(proposedInvalidBlt))
	require.True(t, rr.IsVoted(validBlt))

	needRenewal = is.RemoveOldBallots(proposedInvalidBlt)
	require.False(t, needRenewal) // because it's not made by itself

	require.False(t, rr.IsVoted(proposedInvalidBlt))
	require.True(t, rr.IsVoted(validBlt))

	needRenewal = is.RemoveOldBallots(validBlt)
	require.False(t, needRenewal)

	require.True(t, rr.IsVoted(validBlt))
}
