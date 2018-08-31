// When a node receive ballot, if the ballotState is former then nodeRunnerState,
// the ballot is ignored.
// The node decides it is former or not by NodeRunnerState.

package sebak

import (
	"boscoin.io/sebak/lib/common"
)

type NodeRunnerState struct {
	round       Round
	ballotState sebakcommon.BallotState
}

func NewNodeRunnerState(round Round, ballotState sebakcommon.BallotState) NodeRunnerState {
	p := NodeRunnerState{
		round:       round,
		ballotState: ballotState,
	}

	return p
}
