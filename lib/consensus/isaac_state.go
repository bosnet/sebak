// Holds the ISAACState struct, which is used by
// the ISAACStateManager to handle transition between ballot.
// NodeRunner also holds a copy to efficiently ignore outdated ballot.

package consensus

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
)

type ISAACState struct {
	Round       round.Round
	BallotState common.BallotState
}

func (s ISAACState) IsLater(target ISAACState) (result bool) {
	if s.Round.BlockHeight != target.Round.BlockHeight {
		return s.Round.BlockHeight < target.Round.BlockHeight
	}
	if s.Round.Number != target.Round.Number {
		return s.Round.Number < target.Round.Number
	}
	return s.BallotState < target.BallotState
}
