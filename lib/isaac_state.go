// When a node receive ballot, if the ballotState is former then nodeRunnerState,
// the ballot is ignored.
// The node decides it is former or not by ISAACState.

package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/round"
)

type ISAACState struct {
	round       round.Round
	ballotState common.BallotState
}

func NewISAACState(round round.Round, ballotState common.BallotState) ISAACState {
	p := ISAACState{
		round:       round,
		ballotState: ballotState,
	}

	return p
}
