package consensus

import (
	"testing"

	"boscoin.io/sebak/lib/ballot"
	"github.com/stretchr/testify/require"
)

func pushISAACState(states *[]ISAACState, height uint64, round uint64, ballotState ballot.State) {
	is := ISAACState{
		Height:      height,
		Round:       round,
		BallotState: ballotState,
	}

	*states = append(*states, is)
}

func TestBallotSendRecord(t *testing.T) {
	r := NewBallotSendRecord("n1")

	states := []ISAACState{}

	pushISAACState(&states, 1, 0, ballot.StateINIT)
	pushISAACState(&states, 1, 0, ballot.StateSIGN)
	pushISAACState(&states, 1, 0, ballot.StateACCEPT)

	pushISAACState(&states, 1, 1, ballot.StateINIT)
	pushISAACState(&states, 1, 1, ballot.StateSIGN)

	pushISAACState(&states, 2, 0, ballot.StateINIT)
	pushISAACState(&states, 2, 0, ballot.StateSIGN)
	pushISAACState(&states, 2, 0, ballot.StateACCEPT)
	pushISAACState(&states, 3, 0, ballot.StateINIT)

	require.Equal(t, 9, len(states))
	for _, state := range states {
		r.SetSent(state)
	}

	for _, state := range states {
		require.True(t, r.Sent(state))
	}

	require.True(t, r.Sent(ISAACState{
		Height:      1,
		Round:       0,
		BallotState: ballot.StateINIT,
	}))

	require.False(t, r.Sent(ISAACState{
		Height:      1,
		Round:       1,
		BallotState: ballot.StateACCEPT,
	}))

	r.RemoveLowerThanOrEqualHeight(1)

	require.False(t, r.Sent(ISAACState{
		Height:      1,
		Round:       0,
		BallotState: ballot.StateINIT,
	}))

	r.RemoveLowerThanOrEqualHeight(2)

	require.True(t, r.Sent(ISAACState{
		Height:      3,
		Round:       0,
		BallotState: ballot.StateINIT,
	}))

}
