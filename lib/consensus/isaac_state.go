// Holds the ISAACState struct, which is used by
// the ISAACStateManager to handle transition between ballot.
// NodeRunner also holds a copy to efficiently ignore outdated ballot.

package consensus

import (
	"boscoin.io/sebak/lib/ballot"
)

type ISAACState struct {
	Height      uint64
	Round       uint64
	BallotState ballot.State
}

func (s ISAACState) IsLater(target ISAACState) bool {
	if s.Height != target.Height {
		return s.Height < target.Height
	}
	if s.Round != target.Round {
		return s.Round < target.Round
	}
	return s.BallotState < target.BallotState
}
