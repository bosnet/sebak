// When a node receives a ballot,
// if the ISAACState of the ballot is before than the ISAACState of node,
// the ballot is ignored.

package consensus

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
)

type ISAACState struct {
	Round       round.Round
	BallotState common.BallotState
}

func NewISAACState(round round.Round, ballotState common.BallotState) ISAACState {
	p := ISAACState{
		Round:       round,
		BallotState: ballotState,
	}

	return p
}
