package consensus

import (
	"testing"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/voting"
	"github.com/stretchr/testify/require"
)

func insertRunningRound(t *testing.T, is *ISAAC, source string, proposer string, height uint64, round uint64, ballotState ballot.State, vote voting.Hole) {
	basis := voting.Basis{
		Height: height,
		Round:  round,
	}

	b := *ballot.NewBallot(source, proposer, basis, []string{})
	b.SetVote(ballotState, vote)

	if runningRound, found := is.RunningRounds[basis.Index()]; !found {
		rr, err := NewRunningRound(proposer, b)
		require.NoError(t, err)
		is.RunningRounds[basis.Index()] = rr
	} else {
		runningRound.Vote(b)
	}

}

func TestRunningRoundsLowerOrEqualHeight(t *testing.T) {
	is := ISAAC{
		RunningRounds: map[string]*RunningRound{},
		log:           log.New(),
	}

	nodes := []string{"node1", "node2", "node3"}

	insertRunningRound(t, &is, nodes[0], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[0], 10, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[0], 10, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[0], 10, 0, ballot.StateACCEPT, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 10, 1, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 10, 1, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 10, 1, ballot.StateACCEPT, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)

	require.Equal(t, 3, len(is.RunningRounds))

	is.RemoveRunningRoundsLowerOrEqualHeight(10)
	require.Equal(t, 1, len(is.RunningRounds))

	is.RemoveRunningRoundsLowerOrEqualHeight(9)
	require.Equal(t, 1, len(is.RunningRounds))

	is.RemoveRunningRoundsLowerOrEqualHeight(11)
	require.Equal(t, 0, len(is.RunningRounds))
}

func TestRunningRoundsLowerOrEqualBasis(t *testing.T) {
	is := ISAAC{
		RunningRounds: map[string]*RunningRound{},
		log:           log.New(),
	}

	nodes := []string{"node1", "node2", "node3"}

	insertRunningRound(t, &is, nodes[0], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[0], 10, 0, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[0], 10, 0, ballot.StateACCEPT, voting.NO)
	insertRunningRound(t, &is, nodes[1], nodes[0], 10, 0, ballot.StateACCEPT, voting.NO)
	insertRunningRound(t, &is, nodes[2], nodes[0], 10, 0, ballot.StateACCEPT, voting.NO)

	insertRunningRound(t, &is, nodes[0], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 10, 1, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 10, 1, ballot.StateACCEPT, voting.EXP)
	insertRunningRound(t, &is, nodes[1], nodes[1], 10, 1, ballot.StateACCEPT, voting.EXP)
	insertRunningRound(t, &is, nodes[2], nodes[1], 10, 1, ballot.StateACCEPT, voting.EXP)

	insertRunningRound(t, &is, nodes[0], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 11, 0, ballot.StateSIGN, voting.YES)

	insertRunningRound(t, &is, nodes[0], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[1], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)
	insertRunningRound(t, &is, nodes[2], nodes[1], 11, 0, ballot.StateACCEPT, voting.YES)

	require.Equal(t, 3, len(is.RunningRounds))

	votingBasis := voting.Basis{Height: 10, Round: 0}

	is.RemoveRunningRoundsLowerOrEqualBasis(votingBasis)
	require.Equal(t, 2, len(is.RunningRounds))

	votingBasis.Height = 11

	is.RemoveRunningRoundsLowerOrEqualBasis(votingBasis)
	require.Equal(t, 0, len(is.RunningRounds))
}
