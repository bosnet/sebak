// When a node receive ballot, if the ballotState is former then nodeRunnerState,
// the ballot is ignored.
// The node decides it is former or not by IsaacState.

package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/round"
)

type IsaacState struct {
	round       round.Round
	ballotState common.BallotState
}

func NewIsaacState(round round.Round, ballotState common.BallotState) IsaacState {
	p := IsaacState{
		round:       round,
		ballotState: ballotState,
	}

	return p
}
