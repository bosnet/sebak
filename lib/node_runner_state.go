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
