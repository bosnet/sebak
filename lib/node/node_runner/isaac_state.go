// When a node receives a ballot,
// if the ISAACState of the ballot is before than the ISAACState of node,
// the ballot is ignored.

package node_runner

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
